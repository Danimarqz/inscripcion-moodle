package controllers

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
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
	"github.com/inscripcion-moodle/go-backend/internal/services/email"
	"github.com/inscripcion-moodle/go-backend/internal/services/moodle"
	pdfimport "github.com/inscripcion-moodle/go-backend/internal/services/pdfimport"
)

type contextKey string

const adminContextKey contextKey = "admin-user"

type AdminController struct {
	db           *gorm.DB
	cache        *redis.Client
	auth         *auth.Service
	service      *admin.Service
	pdfImport    pdfImportService
	moodleClient *moodle.Client
	cfg          *config.Config
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

func NewAdminController(db *gorm.DB, cacheClient *redis.Client, authService *auth.Service, cfg *config.Config) *AdminController {
	return &AdminController{
		db:           db,
		cache:        cacheClient,
		auth:         authService,
		service:      admin.New(db),
		pdfImport:    pdfimport.New(db),
		moodleClient: moodle.New(cfg),
		cfg:          cfg,
	}
}

func (h *AdminController) RegisterRoutes(r chi.Router) {
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
		r.Get("/results/emails", h.downloadSubmissionEmails)
		r.Get("/results/emails/list", h.listSubmissionEmailsJSON)
		r.Post("/results/emails/send", h.sendSubmissionEmails)
		r.Post("/moodle/sync-users", h.syncMoodleUsers)
		r.Get("/exams/{exam_id}/results/official", h.listOfficialResults)
		r.Post("/exams/{exam_id}/results/official", h.createOfficialResult)
		r.Post("/exams/{exam_id}/results/import", h.importOfficialResults)
	})
}

func (h *AdminController) createAdmin(w http.ResponseWriter, r *http.Request) {
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

func (h *AdminController) login(w http.ResponseWriter, r *http.Request) {
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

func (h *AdminController) requireAuth(next http.Handler) http.Handler {
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

func (h *AdminController) checkToken(w http.ResponseWriter, r *http.Request) {
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

func (h *AdminController) listExams(w http.ResponseWriter, r *http.Request) {
	exams, err := h.service.ListExams()
	if err != nil {
		http.Error(w, "failed to list exams", http.StatusInternalServerError)
		return
	}
	writeJSON(w, exams)
}

func (h *AdminController) createExam(w http.ResponseWriter, r *http.Request) {
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

func (h *AdminController) updateExam(w http.ResponseWriter, r *http.Request) {
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

func (h *AdminController) deleteExam(w http.ResponseWriter, r *http.Request) {
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

func (h *AdminController) getExam(w http.ResponseWriter, r *http.Request) {
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

func (h *AdminController) listSubmissions(w http.ResponseWriter, r *http.Request) {
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
	moodleSynced := parseOptionalBool(r.URL.Query().Get("moodle_synced"))
	result, err := h.service.ListSubmissions(uint(examID), limit, offset, includeStats, search, orderBy, orderDir, moodleSynced)
	if err != nil {
		http.Error(w, "failed to load submissions", http.StatusInternalServerError)
		return
	}
	writeJSON(w, result)
}

func (h *AdminController) downloadSubmissionEmails(w http.ResponseWriter, r *http.Request) {
	examID, search, orderBy, orderDir, moodleSynced, err := parseSubmissionEmailFilters(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	emails, err := h.service.ListSubmissionEmails(uint(examID), search, orderBy, orderDir, moodleSynced)
	if err != nil {
		http.Error(w, "failed to load submission emails", http.StatusInternalServerError)
		return
	}

	filename := fmt.Sprintf("emails_exam_%d.txt", examID)
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
	if _, err := w.Write([]byte(strings.Join(emails, "\n"))); err != nil {
		log.Printf("failed to write emails response: %v", err)
	}
}

func (h *AdminController) listSubmissionEmailsJSON(w http.ResponseWriter, r *http.Request) {
	examID, search, orderBy, orderDir, moodleSynced, err := parseSubmissionEmailFilters(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	emails, err := h.service.ListSubmissionEmails(uint(examID), search, orderBy, orderDir, moodleSynced)
	if err != nil {
		http.Error(w, "failed to load submission emails", http.StatusInternalServerError)
		return
	}

	writeJSON(w, emails)
}

func (h *AdminController) sendSubmissionEmails(w http.ResponseWriter, r *http.Request) {
	var payload sendSubmissionEmailsRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if payload.ExamID == 0 {
		http.Error(w, "exam_id is required", http.StatusBadRequest)
		return
	}
	body := strings.TrimSpace(payload.Body)
	if body == "" {
		http.Error(w, "email body is required", http.StatusBadRequest)
		return
	}
	if len(payload.Recipients) == 0 {
		http.Error(w, "at least one recipient is required", http.StatusBadRequest)
		return
	}

	attachments, err := decodeAdminEmailAttachments(payload.Attachments)
	if err != nil {
		http.Error(w, fmt.Sprintf("invalid attachments: %v", err), http.StatusBadRequest)
		return
	}

	emails, err := h.service.ListSubmissionEmails(payload.ExamID, payload.Search, payload.OrderBy, payload.OrderDir, payload.MoodleSynced)
	if err != nil {
		http.Error(w, "failed to load submission emails", http.StatusInternalServerError)
		return
	}
	allowed := make(map[string]struct{}, len(emails))
	for _, candidate := range emails {
		allowed[strings.ToLower(strings.TrimSpace(candidate))] = struct{}{}
	}

	var (
		validRecipients []string
		invalid         []string
		seen            = make(map[string]struct{}, len(payload.Recipients))
	)
	for _, raw := range payload.Recipients {
		email := strings.TrimSpace(raw)
		if email == "" {
			continue
		}
		lower := strings.ToLower(email)
		if _, ok := seen[lower]; ok {
			continue
		}
		seen[lower] = struct{}{}
		if _, ok := allowed[lower]; !ok {
			invalid = append(invalid, email)
			continue
		}
		validRecipients = append(validRecipients, email)
	}

	if len(invalid) > 0 {
		http.Error(w, fmt.Sprintf("los siguientes correos no pertenecen al listado filtrado: %s", strings.Join(invalid, ", ")), http.StatusBadRequest)
		return
	}
	if len(validRecipients) == 0 {
		http.Error(w, "no valid recipients to send email", http.StatusBadRequest)
		return
	}

	subject := strings.TrimSpace(payload.Subject)
	if subject == "" {
		subject = "Comunicado oficial"
	}

	toAddress := strings.TrimSpace(h.cfg.AdminEmail)
	if toAddress == "" {
		toAddress = strings.TrimSpace(h.cfg.SMTPUser)
	}
	if toAddress == "" {
		http.Error(w, "admin email not configured", http.StatusInternalServerError)
		return
	}
	if err := email.SendEmail(h.cfg, []string{toAddress}, subject, body, attachments, validRecipients); err != nil {
		log.Printf("failed to send admin emails: %v", err)
		http.Error(w, "failed to send emails", http.StatusInternalServerError)
		return
	}

	writeJSON(w, map[string]int{"sent": len(validRecipients)})
}

func (h *AdminController) updateSubmission(w http.ResponseWriter, r *http.Request) {
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

func (h *AdminController) deleteSubmission(w http.ResponseWriter, r *http.Request) {
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

func (h *AdminController) createOfficialResult(w http.ResponseWriter, r *http.Request) {
	examID, err := strconv.ParseUint(chi.URLParam(r, "exam_id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid exam id", http.StatusBadRequest)
		return
	}

	var payload admin.CreateOfficialResultRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	result, err := h.service.CreateOfficialResult(uint(examID), payload)
	if err != nil {
		switch {
		case errors.Is(err, gorm.ErrRecordNotFound):
			http.Error(w, "exam not found", http.StatusNotFound)
		case errors.Is(err, admin.ErrInvalidOfficialResult):
			http.Error(w, err.Error(), http.StatusBadRequest)
		case errors.Is(err, admin.ErrOfficialResultExists):
			http.Error(w, err.Error(), http.StatusBadRequest)
		default:
			http.Error(w, "failed to create official result", http.StatusInternalServerError)
		}
		return
	}

	writeJSON(w, result)
}

func (h *AdminController) importOfficialResults(w http.ResponseWriter, r *http.Request) {
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

func (h *AdminController) listOfficialResults(w http.ResponseWriter, r *http.Request) {
	examID, err := strconv.ParseUint(chi.URLParam(r, "exam_id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid exam id", http.StatusBadRequest)
		return
	}
	limit := parseLimitParam(r.URL.Query().Get("limit"))
	offset := parseOffsetParam(r.URL.Query().Get("offset"))
	orderBy := sanitizeOfficialResultsOrderBy(r.URL.Query().Get("order_by"))
	orderDir := sanitizeOrderDir(r.URL.Query().Get("order_dir"))
	results, err := h.service.ListOfficialResults(uint(examID), limit, offset, orderBy, orderDir)
	if err != nil {
		http.Error(w, "failed to load official results", http.StatusInternalServerError)
		return
	}
	writeJSON(w, results)
}

func (h *AdminController) syncMoodleUsers(w http.ResponseWriter, r *http.Request) {
	if h.moodleClient == nil {
		http.Error(w, "moodle not configured", http.StatusServiceUnavailable)
		return
	}

	var users []models.ExamUser
	if err := h.db.Select("id", "email", "dni").Where("moodle_id IS NULL").Find(&users).Error; err != nil {
		http.Error(w, "failed to load exam users", http.StatusInternalServerError)
		return
	}

	if len(users) == 0 {
		writeJSON(w, map[string]string{"status": "no users to sync"})
		return
	}

	go h.syncMoodleUsersAsync(append([]models.ExamUser(nil), users...))

	writeJSON(w, map[string]string{"status": "sync started"})
}

func (h *AdminController) syncMoodleUsersAsync(users []models.ExamUser) {
	ctx, cancel := context.WithTimeout(context.Background(), moodleAdminSyncTimeout)
	defer cancel()

	enrolledUsers, err := h.moodleClient.GetEnrolledUsers(ctx, moodle.MoodleExamCourseID, nil)
	if err != nil {
		log.Printf("failed to fetch enrolled users for course %d: %v", moodle.MoodleExamCourseID, err)
		return
	}

	enrolledByEmail := make(map[string]moodle.EnrolledUser, len(enrolledUsers))
	for _, enrolled := range enrolledUsers {
		email := strings.ToLower(strings.TrimSpace(enrolled.Email))
		if email == "" {
			continue
		}
		enrolledByEmail[email] = enrolled
	}

	checked := len(users)
	synced := 0
	failed := 0

	for _, user := range users {
		email := strings.ToLower(strings.TrimSpace(user.Email))
		if email == "" {
			log.Printf("moodle sync for %s (%s) skipped: missing email", user.Email, user.DNI)
			failed++
			continue
		}

		enrolled, ok := enrolledByEmail[email]
		if !ok {
			log.Printf("moodle sync for %s (%s) skipped: not enrolled in course %d", user.Email, user.DNI, moodle.MoodleExamCourseID)
			failed++
			continue
		}

		if err := h.db.Model(&models.ExamUser{}).
			Where("id = ?", user.ID).
			Where("moodle_id IS NULL").
			Update("moodle_id", enrolled.ID).Error; err != nil {
			log.Printf("moodle sync for %s (%s) failed: %v", user.Email, user.DNI, err)
			failed++
			continue
		}
		synced++
	}

	log.Printf("moodle sync finished: checked=%d synced=%d failed=%d", checked, synced, failed)
}

func (h *AdminController) invalidateExamCaches(examID uint) {
	if h.cache == nil {
		return
	}
	cache.InvalidateExams(h.cache)
	cache.InvalidateQuestions(h.cache, examID)
	cache.InvalidateCheckCache(h.cache, examID)
}

func (h *AdminController) handleAdminError(w http.ResponseWriter, err error) {
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

func parseOptionalBool(raw string) *bool {
	if raw == "" {
		return nil
	}
	value, err := strconv.ParseBool(strings.ToLower(strings.TrimSpace(raw)))
	if err != nil {
		return nil
	}
	return &value
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

func sanitizeOfficialResultsOrderBy(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "dni":
		return "dni"
	case "nombre":
		return "nombre"
	case "apellidos":
		return "apellidos"
	case "usuario":
		return "usuario"
	case "creado", "created_at":
		return "creado"
	default:
		return "apellidos"
	}
}

type sendSubmissionEmailsRequest struct {
	ExamID       uint                          `json:"exam_id"`
	Subject      string                        `json:"subject"`
	Body         string                        `json:"body"`
	Recipients   []string                      `json:"recipients"`
	Search       string                        `json:"search"`
	OrderBy      string                        `json:"order_by"`
	OrderDir     string                        `json:"order_dir"`
	MoodleSynced *bool                         `json:"moodle_synced"`
	Attachments  []adminEmailAttachmentRequest `json:"attachments"`
}

type adminEmailAttachmentRequest struct {
	Filename    string `json:"filename"`
	Content     string `json:"content"`
	ContentType string `json:"content_type"`
}

func parseSubmissionEmailFilters(r *http.Request) (uint, string, string, string, *bool, error) {
	examIDStr := r.URL.Query().Get("exam_id")
	if examIDStr == "" {
		return 0, "", "", "", nil, fmt.Errorf("exam_id required")
	}
	examID, err := strconv.ParseUint(examIDStr, 10, 64)
	if err != nil {
		return 0, "", "", "", nil, fmt.Errorf("invalid exam id")
	}
	search := strings.TrimSpace(r.URL.Query().Get("search"))
	orderBy := sanitizeOrderBy(r.URL.Query().Get("order_by"))
	orderDir := sanitizeOrderDir(r.URL.Query().Get("order_dir"))
	moodleSynced := parseOptionalBool(r.URL.Query().Get("moodle_synced"))
	return uint(examID), search, orderBy, orderDir, moodleSynced, nil
}

func decodeAdminEmailAttachments(items []adminEmailAttachmentRequest) ([]email.Attachment, error) {
	if len(items) == 0 {
		return nil, nil
	}
	attachments := make([]email.Attachment, 0, len(items))
	for _, item := range items {
		if strings.TrimSpace(item.Filename) == "" {
			return nil, fmt.Errorf("attachment filename is required")
		}
		content, err := decodeBase64Payload(item.Content)
		if err != nil {
			return nil, err
		}
		attachments = append(attachments, email.Attachment{
			Filename:    strings.TrimSpace(item.Filename),
			Content:     content,
			ContentType: strings.TrimSpace(item.ContentType),
		})
	}
	return attachments, nil
}

func decodeBase64Payload(raw string) ([]byte, error) {
	if raw == "" {
		return nil, fmt.Errorf("attachment content is empty")
	}
	if idx := strings.Index(raw, ","); idx != -1 && strings.HasPrefix(raw[:idx], "data:") {
		raw = raw[idx+1:]
	}
	decoded, err := base64.StdEncoding.DecodeString(raw)
	if err != nil {
		return nil, err
	}
	return decoded, nil
}
