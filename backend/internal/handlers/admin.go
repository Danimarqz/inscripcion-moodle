package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"github.com/inscripcion-moodle/go-backend/internal/cache"
	"github.com/inscripcion-moodle/go-backend/internal/config"
	"github.com/inscripcion-moodle/go-backend/internal/models"
	"github.com/inscripcion-moodle/go-backend/internal/services/admin"
	"github.com/inscripcion-moodle/go-backend/internal/services/auth"
	"github.com/inscripcion-moodle/go-backend/internal/services/moodle"
	pdfimport "github.com/inscripcion-moodle/go-backend/internal/services/pdfimport"
)

type contextKey string

const adminContextKey contextKey = "admin-user"

type AdminHandler struct {
	db           *gorm.DB
	cache        *redis.Client
	auth         *auth.Service
	service      *admin.Service
	pdfImport    pdfImportService
	moodleClient *moodle.Client
}

type pdfImportService interface {
	ImportOfficialResultsPDF(context.Context, uint, string, bool) (*pdfimport.PDFImportResult, error)
}

const (
	defaultSubmissionsLimit = 100
	maxSubmissionsLimit     = 500
	moodleAdminSyncTimeout  = 20 * time.Second
)

type adminRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type tokenResponse struct {
	AccessToken string `json:"access_token"`
}

func NewAdminHandler(db *gorm.DB, cacheClient *redis.Client, authService *auth.Service, cfg *config.Config) *AdminHandler {
	return &AdminHandler{
		db:           db,
		cache:        cacheClient,
		auth:         authService,
		service:      admin.New(db),
		pdfImport:    pdfimport.New(db),
		moodleClient: moodle.New(cfg),
	}
}

func (h *AdminHandler) RegisterRoutes(r chi.Router) {
	r.Post("/create-admin", h.createAdmin)
	r.Post("/login", h.login)
	r.Group(func(r chi.Router) {
		r.Use(h.requireAuth)
		r.Get("/check-token", h.checkToken)
		r.Get("/exams", h.listExams)
		r.Post("/exams", h.createExam)
		r.Put("/exams/{exam_id}", h.updateExam)
		r.Delete("/exams/{exam_id}/delete", h.deleteExam)
		r.Get("/exams/{exam_id}", h.getExam)
		r.Get("/results", h.listSubmissions)
		r.Put("/results/{submission_id}", h.updateSubmission)
		r.Delete("/results/{submission_id}", h.deleteSubmission)
		r.Post("/moodle/sync-users", h.syncMoodleUsers)
		r.Get("/exams/{exam_id}/results/official", h.listOfficialResults)
		r.Post("/exams/{exam_id}/results/import", h.importOfficialResults)
	})
}

func (h *AdminHandler) createAdmin(w http.ResponseWriter, r *http.Request) {
	var payload adminRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	var existing int64
	if err := h.db.Model(&models.AdminUser{}).Count(&existing).Error; err != nil {
		http.Error(w, "failed to verify existing admin", http.StatusInternalServerError)
		return
	}
	if existing > 0 {
		http.Error(w, "administrador ya existe", http.StatusForbidden)
		return
	}

	hash, err := h.auth.HashPassword(payload.Password)
	if err != nil {
		http.Error(w, "failed to hash password", http.StatusInternalServerError)
		return
	}

	admin := models.AdminUser{
		Username:     payload.Username,
		PasswordHash: hash,
	}
	if err := h.db.Create(&admin).Error; err != nil {
		http.Error(w, "failed to create admin user", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	_, _ = w.Write([]byte(`{"message":"Administrador creado con exito"}`))
}

func (h *AdminHandler) login(w http.ResponseWriter, r *http.Request) {
	var payload adminRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	var admin models.AdminUser
	if err := h.db.Where("username = ?", payload.Username).First(&admin).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			http.Error(w, "credenciales invalidas", http.StatusUnauthorized)
			return
		}
		http.Error(w, "failed to lookup admin", http.StatusInternalServerError)
		return
	}

	if !h.auth.VerifyPassword(admin.PasswordHash, payload.Password) {
		http.Error(w, "credenciales invalidas", http.StatusUnauthorized)
		return
	}

	token, err := h.auth.CreateToken(admin.Username)
	if err != nil {
		http.Error(w, "failed to create token", http.StatusInternalServerError)
		return
	}

	writeJSON(w, tokenResponse{AccessToken: token})
}

func (h *AdminHandler) requireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "missing authorization header", http.StatusUnauthorized)
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			http.Error(w, "invalid authorization header format", http.StatusUnauthorized)
			return
		}

		username, err := h.auth.ParseToken(parts[1])
		if err != nil {
			http.Error(w, "invalid token", http.StatusUnauthorized)
			return
		}

		var admin models.AdminUser
		if err := h.db.Where("username = ?", username).First(&admin).Error; err != nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), adminContextKey, &admin)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (h *AdminHandler) checkToken(w http.ResponseWriter, r *http.Request) {
	admin := adminFromContext(r.Context())
	if admin == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	writeJSON(w, map[string]string{
		"detail": "Token valido",
		"user":   admin.Username,
	})
}

func (h *AdminHandler) listExams(w http.ResponseWriter, r *http.Request) {
	exams, err := h.service.ListExams()
	if err != nil {
		http.Error(w, "failed to list exams", http.StatusInternalServerError)
		return
	}
	writeJSON(w, exams)
}

func (h *AdminHandler) createExam(w http.ResponseWriter, r *http.Request) {
	var req admin.CreateExamRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	exam, err := h.service.CreateExam(req)
	if err != nil {
		h.handleAdminError(w, err)
		return
	}
	h.invalidateExamCaches(exam.ID)
	writeJSON(w, exam)
}

func (h *AdminHandler) updateExam(w http.ResponseWriter, r *http.Request) {
	examID, err := strconv.ParseUint(chi.URLParam(r, "exam_id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid exam id", http.StatusBadRequest)
		return
	}
	var req admin.EditExamRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	exam, err := h.service.UpdateExam(uint(examID), req)
	if err != nil {
		h.handleAdminError(w, err)
		return
	}
	h.invalidateExamCaches(exam.ID)
	writeJSON(w, exam)
}

func (h *AdminHandler) deleteExam(w http.ResponseWriter, r *http.Request) {
	examID, err := strconv.ParseUint(chi.URLParam(r, "exam_id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid exam id", http.StatusBadRequest)
		return
	}
	if err := h.service.DeleteExam(uint(examID)); err != nil {
		http.Error(w, "failed to delete exam", http.StatusInternalServerError)
		return
	}
	h.invalidateExamCaches(uint(examID))
	writeJSON(w, map[string]string{"detail": "Examen eliminado correctamente"})
}

func (h *AdminHandler) getExam(w http.ResponseWriter, r *http.Request) {
	examID, err := strconv.ParseUint(chi.URLParam(r, "exam_id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid exam id", http.StatusBadRequest)
		return
	}
	exam, err := h.service.GetExam(uint(examID))
	if err != nil {
		http.Error(w, "exam not found", http.StatusNotFound)
		return
	}
	writeJSON(w, exam)
}

func (h *AdminHandler) listSubmissions(w http.ResponseWriter, r *http.Request) {
	examIDStr := r.URL.Query().Get("exam_id")
	if examIDStr == "" {
		http.Error(w, "exam_id required", http.StatusBadRequest)
		return
	}
	examID, err := strconv.ParseUint(examIDStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid exam id", http.StatusBadRequest)
		return
	}
	limit := parseLimitParam(r.URL.Query().Get("limit"))
	offset := parseOffsetParam(r.URL.Query().Get("offset"))
	includeStats := parseBoolParam(r.URL.Query().Get("first_load"), true)
	search := strings.TrimSpace(r.URL.Query().Get("search"))
	orderBy := sanitizeOrderBy(r.URL.Query().Get("order_by"))
	orderDir := sanitizeOrderDir(r.URL.Query().Get("order_dir"))
	result, err := h.service.ListSubmissions(uint(examID), limit, offset, includeStats, search, orderBy, orderDir)
	if err != nil {
		http.Error(w, "failed to load submissions", http.StatusInternalServerError)
		return
	}
	writeJSON(w, result)
}

func (h *AdminHandler) updateSubmission(w http.ResponseWriter, r *http.Request) {
	submissionIDStr := chi.URLParam(r, "submission_id")
	submissionID, err := strconv.ParseUint(submissionIDStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid submission id", http.StatusBadRequest)
		return
	}
	var payload admin.SubmissionUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	updated, err := h.service.UpdateSubmission(uint(submissionID), payload)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			http.Error(w, "Intento no encontrado", http.StatusNotFound)
			return
		}
		http.Error(w, "failed to update submission", http.StatusInternalServerError)
		return
	}
	writeJSON(w, updated)
}

func (h *AdminHandler) deleteSubmission(w http.ResponseWriter, r *http.Request) {
	submissionID, err := strconv.ParseUint(chi.URLParam(r, "submission_id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid submission id", http.StatusBadRequest)
		return
	}
	examID, err := h.service.DeleteSubmission(uint(submissionID))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			http.Error(w, "Intento no encontrado", http.StatusNotFound)
			return
		}
		http.Error(w, "failed to delete submission", http.StatusInternalServerError)
		return
	}
	h.invalidateExamCaches(examID)
	writeJSON(w, map[string]string{"detail": "Intento eliminado correctamente"})
}

func (h *AdminHandler) importOfficialResults(w http.ResponseWriter, r *http.Request) {
	examID, err := strconv.ParseUint(chi.URLParam(r, "exam_id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid exam id", http.StatusBadRequest)
		return
	}
	if err := r.ParseMultipartForm(32 << 20); err != nil && err != http.ErrNotMultipart {
		http.Error(w, "failed to parse multipart data", http.StatusBadRequest)
		return
	}
	file, _, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "missing pdf file", http.StatusBadRequest)
		return
	}
	defer func() {
		_ = file.Close()
	}()

	tmp, err := os.CreateTemp("", "official_results_*.pdf")
	if err != nil {
		http.Error(w, "failed to create temp file", http.StatusInternalServerError)
		return
	}
	defer func() {
		_ = os.Remove(tmp.Name())
	}()
	defer func() {
		_ = tmp.Close()
	}()

	if _, err := io.Copy(tmp, file); err != nil {
		http.Error(w, "failed to store pdf", http.StatusInternalServerError)
		return
	}

	replace := parseBoolParam(r.URL.Query().Get("replace_existing"), true)
	report, err := h.pdfImport.ImportOfficialResultsPDF(r.Context(), uint(examID), tmp.Name(), replace)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, report)
}

func (h *AdminHandler) listOfficialResults(w http.ResponseWriter, r *http.Request) {
	examID, err := strconv.ParseUint(chi.URLParam(r, "exam_id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid exam id", http.StatusBadRequest)
		return
	}
	results, err := h.service.ListOfficialResults(uint(examID))
	if err != nil {
		http.Error(w, "failed to load official results", http.StatusInternalServerError)
		return
	}
	writeJSON(w, results)
}

func (h *AdminHandler) syncMoodleUsers(w http.ResponseWriter, r *http.Request) {
	if h.moodleClient == nil {
		http.Error(w, "moodle not configured", http.StatusServiceUnavailable)
		return
	}

	var users []models.ExamUser
	if err := h.db.Select("id", "email", "dni").Where("moodle_id IS NULL").Find(&users).Error; err != nil {
		http.Error(w, "failed to load exam users", http.StatusInternalServerError)
		return
	}

	checked := len(users)
	synced := 0
	failed := 0

	for _, user := range users {
		ctx, cancel := context.WithTimeout(r.Context(), moodleAdminSyncTimeout)
		err := moodle.SyncExamUser(ctx, h.db, h.moodleClient, user.Email, user.DNI)
		cancel()
		if err != nil {
			log.Printf("moodle sync for %s (%s) failed: %v", user.Email, user.DNI, err)
			failed++
			continue
		}
		synced++
	}

	writeJSON(w, map[string]int{
		"checked": checked,
		"synced":  synced,
		"failed":  failed,
	})
}

func (h *AdminHandler) invalidateExamCaches(examID uint) {
	if h.cache == nil {
		return
	}
	cache.InvalidateExams(h.cache)
	cache.InvalidateQuestions(h.cache, examID)
	cache.InvalidateCheckCache(h.cache, examID)
}

func (h *AdminHandler) handleAdminError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, admin.ErrExamNameConflict), errors.Is(err, admin.ErrQuestionNotFound):
		http.Error(w, err.Error(), http.StatusBadRequest)
	case errors.Is(err, admin.ErrActiveQuestions):
		http.Error(w, err.Error(), http.StatusBadRequest)
	default:
		http.Error(w, "operation failed", http.StatusInternalServerError)
	}
}

func adminFromContext(ctx context.Context) *models.AdminUser {
	val, _ := ctx.Value(adminContextKey).(*models.AdminUser)
	return val
}

func writeJSON(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(data)
}

func parseLimitParam(raw string) int {
	if raw == "" {
		return defaultSubmissionsLimit
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		return defaultSubmissionsLimit
	}
	if value > maxSubmissionsLimit {
		return maxSubmissionsLimit
	}
	return value
}

func parseOffsetParam(raw string) int {
	if raw == "" {
		return 0
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < 0 {
		return 0
	}
	return value
}

func parseBoolParam(raw string, def bool) bool {
	if raw == "" {
		return def
	}
	value, err := strconv.ParseBool(strings.ToLower(raw))
	if err != nil {
		return def
	}
	return value
}

func sanitizeOrderBy(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "score":
		return "score"
	case "name":
		return "name"
	case "surname":
		return "surname"
	case "submitted_at", "time":
		return "submitted_at"
	default:
		return "submitted_at"
	}
}

func sanitizeOrderDir(raw string) string {
	dir := strings.ToLower(strings.TrimSpace(raw))
	if dir == "asc" {
		return "asc"
	}
	return "desc"
}
