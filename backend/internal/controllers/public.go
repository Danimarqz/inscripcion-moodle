package controllers

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"github.com/inscripcion-moodle/go-backend/internal/cache"
	"github.com/inscripcion-moodle/go-backend/internal/config"
	"github.com/inscripcion-moodle/go-backend/internal/constants"
	"github.com/inscripcion-moodle/go-backend/internal/helpers"
	"github.com/inscripcion-moodle/go-backend/internal/models"
	"github.com/inscripcion-moodle/go-backend/internal/repository"
	"github.com/inscripcion-moodle/go-backend/internal/services/cloudfrontsign"
	examservice "github.com/inscripcion-moodle/go-backend/internal/services/exam"
	"github.com/inscripcion-moodle/go-backend/internal/services/feedback"
	"github.com/inscripcion-moodle/go-backend/internal/services/moodle"
)

const (
	examsCacheKey        = "public:exams:v2"
	questionsCachePrefix = "public:questions:v2"
	moodleSyncTimeout    = 15 * time.Second
)

type PublicController struct {
	db           *gorm.DB
	cache        *cache.Cache
	cacheTTL     time.Duration
	moodleClient *moodle.Client
	service      *examservice.Service
	feedback     *feedback.Service
}

func NewPublicController(db *gorm.DB, rds *redis.Client, cfg *config.Config, service *examservice.Service) *PublicController {
	var signer *cloudfrontsign.Signer
	var signerErr string
	if cfg.CloudFrontDomain != "" && cfg.CloudFrontKeyPairID != "" && cfg.CloudFrontPrivateKey != "" {
		var err error
		signer, err = cloudfrontsign.New(cfg.CloudFrontDomain, cfg.CloudFrontKeyPairID, cfg.CloudFrontPrivateKey)
		if err != nil {
			signerErr = err.Error()
			log.Printf("cloudfrontsign: failed to initialize signer: %v", err)
		}
	} else {
		signerErr = fmt.Sprintf("missing env vars (domain=%q keypair=%q key_len=%d)",
			cfg.CloudFrontDomain, cfg.CloudFrontKeyPairID, len(cfg.CloudFrontPrivateKey))
		log.Printf("cloudfrontsign: %s", signerErr)
	}

	c := cache.New(rds)
	feedbackService := feedback.New(
		db, c, signer, cfg.CloudFrontDomain, signerErr,
		repository.NewSubmissionRepository(),
		repository.NewQuestionRepository(),
	)

	return &PublicController{
		db:           db,
		cache:        c,
		cacheTTL:     cfg.PublicCacheTTL,
		moodleClient: moodle.New(cfg),
		service:      service,
		feedback:     feedbackService,
	}
}

// cachedActiveExams returns the JSON payload of all active exams (with slugs),
// served from Redis. Both the exam list and exam-by-slug read from this single
// cached payload, so neither hits the DB per request.
func (h *PublicController) cachedActiveExams(ctx context.Context) ([]byte, error) {
	return h.cache.GetOrSet(ctx, examsCacheKey, h.cacheTTL, func() ([]byte, error) {
		var exams []models.Exam
		if err := h.db.Where("is_active = ?", true).Find(&exams).Error; err != nil {
			return nil, err
		}
		for i := range exams {
			exams[i].Slug = helpers.CreateSlug(exams[i].Name)
		}
		return json.Marshal(exams)
	})
}

func (h *PublicController) GetExams(w http.ResponseWriter, r *http.Request) {
	payload, err := h.cachedActiveExams(r.Context())
	if err != nil {
		http.Error(w, constants.FailedToLoadExams, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(payload)
}

func (h *PublicController) SubmitExam(w http.ResponseWriter, r *http.Request) {
	defer func() {
		if err := r.Body.Close(); err != nil {
			log.Printf("failed to close request body: %v", err)
		}
	}()
	var req examservice.SubmitExamRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, constants.InvalidRequest, http.StatusBadRequest)
		return
	}

	if strings.TrimSpace(req.Email) == "" || strings.TrimSpace(req.DNI) == "" {
		http.Error(w, constants.EmailAndDNIAreRequired, http.StatusBadRequest)
		return
	}

	payload, err := h.service.ProcessExamSubmission(req)
	if err != nil {
		h.handleError(w, err)
		return
	}

	h.scheduleMoodleSync(req.Email, req.DNI)

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		log.Printf("failed to encode response: %v", err)
	}
}

func (h *PublicController) GetQuestionStubs(w http.ResponseWriter, r *http.Request) {
	examIDParam := chi.URLParam(r, "exam_id")
	examID, err := strconv.ParseUint(examIDParam, 10, 64)
	if err != nil {
		http.Error(w, constants.InvalidExamID, http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	key := h.questionsCacheKey(uint(examID))

	payload, err := h.cache.GetOrSet(ctx, key, h.cacheTTL, func() ([]byte, error) {
		stubs, err := h.service.GetQuestionStubs(ctx, uint(examID))
		if err != nil {
			return nil, err
		}
		return json.Marshal(stubs)
	})

	if err != nil {
		if errors.Is(err, examservice.ErrExamNotFound) {
			http.Error(w, constants.ExamNotFound, http.StatusNotFound)
			return
		}
		http.Error(w, constants.FailedToLoadQuestions, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(payload)
}

func (h *PublicController) GetExamBySlug(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	if slug == "" {
		http.Error(w, "slug is required", http.StatusBadRequest)
		return
	}

	payload, err := h.cachedActiveExams(r.Context())
	if err != nil {
		http.Error(w, constants.FailedToLoadExams, http.StatusInternalServerError)
		return
	}

	var exams []models.Exam
	if err := json.Unmarshal(payload, &exams); err != nil {
		http.Error(w, constants.FailedToLoadExams, http.StatusInternalServerError)
		return
	}
	for i := range exams {
		if exams[i].Slug == slug {
			w.Header().Set("Content-Type", "application/json")
			if err := json.NewEncoder(w).Encode(exams[i]); err != nil {
				log.Printf("failed to encode response: %v", err)
			}
			return
		}
	}
	h.handleError(w, examservice.ErrExamNotFound)
}

func (h *PublicController) CheckSubmission(w http.ResponseWriter, r *http.Request) {
	defer func() {
		if err := r.Body.Close(); err != nil {
			log.Printf("failed to close request body: %v", err)
		}
	}()
	var req examservice.SubmissionCheckRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, constants.InvalidRequest, http.StatusBadRequest)
		return
	}

	if strings.TrimSpace(req.Email) == "" || strings.TrimSpace(req.DNI) == "" {
		http.Error(w, constants.EmailAndDNIAreRequired, http.StatusBadRequest)
		return
	}

	normalizedEmail := strings.ToLower(strings.TrimSpace(req.Email))
	normalizedDNI := examservice.NormalizeDNI(req.DNI)
	cacheKeyHash := sha256.Sum256(fmt.Appendf(nil, "%d:%s:%s", req.ExamID, normalizedEmail, normalizedDNI))
	cacheKey := fmt.Sprintf("check_sub:%x", cacheKeyHash)

	if cachedData, ok := h.cache.Get(r.Context(), cacheKey); ok {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(cachedData)
		return
	}

	payload, err := h.service.BuildSubmissionCheckResponse(req)
	if err != nil {
		h.handleError(w, err)
		return
	}

	respBytes, err := json.Marshal(payload)
	if err != nil {
		log.Printf("failed to encode response: %v", err)
		http.Error(w, constants.ErrorInterno, http.StatusInternalServerError)
		return
	}

	h.cache.Set(r.Context(), cacheKey, respBytes, 1*time.Minute)

	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(respBytes)
}

func (h *PublicController) CheckOfficialResultMatch(w http.ResponseWriter, r *http.Request) {
	defer func() {
		if err := r.Body.Close(); err != nil {
			log.Printf("failed to close request body: %v", err)
		}
	}()
	var req examservice.OfficialResultMatchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, constants.InvalidRequest, http.StatusBadRequest)
		return
	}

	if strings.TrimSpace(req.DNI) == "" {
		http.Error(w, constants.DNIRequired, http.StatusBadRequest)
		return
	}

	normalizedName := examservice.NormalizeDNI(req.Name)
	normalizedSurname := examservice.NormalizeDNI(req.Surname)
	normalizedDNI := examservice.NormalizeDNI(req.DNI)
	cacheKeyHash := sha256.Sum256(fmt.Appendf(nil, "%d:%s:%s:%s", req.ExamID, normalizedName, normalizedSurname, normalizedDNI))
	cacheKey := fmt.Sprintf("match_oficial:%x", cacheKeyHash)

	if cachedData, ok := h.cache.Get(r.Context(), cacheKey); ok {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(cachedData)
		return
	}

	match, err := examservice.CheckOfficialResultMatch(h.db, req)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, examservice.ErrExamNotFound) {
			status = http.StatusNotFound
		}
		http.Error(w, constants.FailedToVerifyResult, status)
		return
	}

	respPayload := examservice.OfficialResultMatchResponse{Match: match}
	respBytes, err := json.Marshal(respPayload)
	if err != nil {
		log.Printf("failed to encode response: %v", err)
		http.Error(w, constants.ErrorInterno, http.StatusInternalServerError)
		return
	}

	h.cache.Set(r.Context(), cacheKey, respBytes, 10*time.Minute)

	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(respBytes)
}

func (h *PublicController) UpdateMerits(w http.ResponseWriter, r *http.Request) {
	defer func() {
		if err := r.Body.Close(); err != nil {
			log.Printf("failed to close request body: %v", err)
		}
	}()
	var req examservice.UpdateMeritsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, constants.InvalidRequest, http.StatusBadRequest)
		return
	}

	if strings.TrimSpace(req.Email) == "" || strings.TrimSpace(req.DNI) == "" {
		http.Error(w, constants.EmailAndDNIAreRequired, http.StatusBadRequest)
		return
	}

	resp, err := h.service.UpdateMerits(req)
	if err != nil {
		h.handleError(w, err)
		return
	}

	// Invalidate check_submission cache for this user/exam
	normalizedEmail := strings.ToLower(strings.TrimSpace(req.Email))
	normalizedDNI := examservice.NormalizeDNI(req.DNI)
	cacheKeyHash := sha256.Sum256(fmt.Appendf(nil, "%d:%s:%s", req.ExamID, normalizedEmail, normalizedDNI))
	cacheKey := fmt.Sprintf("check_sub:%x", cacheKeyHash)
	h.cache.Del(r.Context(), cacheKey)

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Printf("failed to encode response: %v", err)
	}
}

func (h *PublicController) questionsCacheKey(examID uint) string {
	return fmt.Sprintf("%s:%d", questionsCachePrefix, examID)
}

func (h *PublicController) handleError(w http.ResponseWriter, err error) {
	status := http.StatusBadRequest
	switch {
	case errors.Is(err, examservice.ErrSubmissionNotFound), errors.Is(err, examservice.ErrExamNotFound):
		status = http.StatusNotFound
	case errors.Is(err, examservice.ErrExamNotActive), errors.Is(err, examservice.ErrResultsNotViewable):
		status = http.StatusForbidden
	case errors.Is(err, examservice.ErrOfficialResultMissing):
		status = http.StatusForbidden
	case errors.Is(err, examservice.ErrNotPassed):
		status = http.StatusForbidden
	}
	h.writeJSONError(w, status, err.Error())
}

func (h *PublicController) scheduleMoodleSync(email, dni string) {
	if h.moodleClient == nil {
		return
	}
	if err := moodle.EnqueueSyncUser(h.db, h.moodleClient, email, dni); err != nil {
		log.Printf("failed to enqueue moodle sync for %s / %s: %v", email, dni, err)
	}
}

func (h *PublicController) writeJSONError(w http.ResponseWriter, status int, detail string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"detail": detail})
}
