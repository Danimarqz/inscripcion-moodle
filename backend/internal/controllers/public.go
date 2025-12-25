package controllers

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
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
	"github.com/inscripcion-moodle/go-backend/internal/models"
	"github.com/inscripcion-moodle/go-backend/internal/repository"
	examservice "github.com/inscripcion-moodle/go-backend/internal/services/exam"
	"github.com/inscripcion-moodle/go-backend/internal/services/moodle"
)

const (
	examsCacheKey         = "public:exams"
	questionsCachePrefix  = "public:questions"
	submissionCachePrefix = "public:check"
	moodleSyncTimeout     = 15 * time.Second
)

type QuestionStub struct {
	ID          uint `json:"id"`
	Name        int  `json:"name"`
	IsActive    bool `json:"is_active"`
	IsCancelled bool `json:"is_cancelled"`
}

type PublicController struct {
	db           *gorm.DB
	cache        *cache.Cache
	cacheTTL     time.Duration
	moodleClient *moodle.Client
}

func NewPublicController(db *gorm.DB, rds *redis.Client, cfg *config.Config) *PublicController {
	return &PublicController{
		db:           db,
		cache:        cache.New(rds),
		cacheTTL:     cfg.PublicCacheTTL,
		moodleClient: moodle.New(cfg),
	}
}

func (h *PublicController) GetExams(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	payload, err := h.cache.GetOrSet(ctx, examsCacheKey, 0, func() ([]byte, error) {
		var exams []models.Exam
		if err := h.db.Where("is_active = ?", true).Find(&exams).Error; err != nil {
			return nil, err
		}
		return json.Marshal(exams)
	})
	if err != nil {
		http.Error(w, constants.FailedToLoadExams, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(payload)
}

func (h *PublicController) SubmitExam(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	var req examservice.SubmitExamRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, constants.InvalidRequest, http.StatusBadRequest)
		return
	}

	if strings.TrimSpace(req.Email) == "" || strings.TrimSpace(req.DNI) == "" {
		http.Error(w, constants.EmailAndDNIAreRequired, http.StatusBadRequest)
		return
	}

	examService := examservice.NewService(h.db, repository.NewExamRepository())
	payload, err := examService.ProcessExamSubmission(req)
	if err != nil {
		h.handleError(w, err)
		return
	}

	h.invalidateCheckCacheForExam(req.ExamID)
	h.scheduleMoodleSync(req.Email, req.DNI)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(payload)
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

	// Service is instantiated here as it's lightweight.
	// For a larger application, it might be part of the controller's dependencies.
	examService := examservice.NewService(h.db, repository.NewExamRepository())

	payload, err := h.cache.GetOrSet(ctx, key, 0, func() ([]byte, error) {
		stubs, err := examService.GetQuestionStubs(ctx, uint(examID))
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

func (h *PublicController) CheckSubmission(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	var req examservice.SubmissionCheckRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, constants.InvalidRequest, http.StatusBadRequest)
		return
	}

	if strings.TrimSpace(req.Email) == "" || strings.TrimSpace(req.DNI) == "" {
		http.Error(w, constants.EmailAndDNIAreRequired, http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	cacheKey := h.submissionCacheKey(req.ExamID, req.Email, req.DNI)

	payload, err := h.cache.GetOrSet(ctx, cacheKey, h.cacheTTL, func() ([]byte, error) {
		payload, err := examservice.BuildSubmissionCheckResponse(h.db, req)
		if err != nil {
			return nil, err
		}
		return json.Marshal(payload)
	})

	if err != nil {
		h.handleError(w, err)
		return
	}

	submissionSetKey := h.submissionSetKey(req.ExamID)
	h.cache.SAdd(ctx, submissionSetKey, cacheKey)

	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(payload)
}

func (h *PublicController) CheckOfficialResultMatch(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	var req examservice.OfficialResultMatchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, constants.InvalidRequest, http.StatusBadRequest)
		return
	}

	if strings.TrimSpace(req.DNI) == "" {
		http.Error(w, constants.DNIRequired, http.StatusBadRequest)
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

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(examservice.OfficialResultMatchResponse{Match: match})
}

func (h *PublicController) questionsCacheKey(examID uint) string {
	return fmt.Sprintf("%s:%d", questionsCachePrefix, examID)
}

func (h *PublicController) submissionCacheKey(examID uint, email, dni string) string {
	normalizedEmail := strings.ToLower(strings.TrimSpace(email))
	normalizedDNI := strings.ToUpper(strings.TrimSpace(dni))
	fingerprint := fmt.Sprintf("%d:%s:%s", examID, normalizedEmail, normalizedDNI)
	sum := sha256.Sum256([]byte(fingerprint))
	return fmt.Sprintf("%s:%d:%s", submissionCachePrefix, examID, hex.EncodeToString(sum[:]))
}

func (h *PublicController) submissionSetKey(examID uint) string {
	return fmt.Sprintf("%s:%d:set", submissionCachePrefix, examID)
}

func (h *PublicController) invalidateCheckCacheForExam(examID uint) {
	if h.cache == nil {
		return
	}
	ctx := context.Background()
	setKey := h.submissionSetKey(examID)
	keys, err := h.cache.SMembers(ctx, setKey)
	if err != nil {
		return
	}
	for _, key := range keys {
		_ = h.cache.Del(ctx, key)
	}
	_ = h.cache.Del(ctx, setKey)
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
	}
	h.writeJSONError(w, status, err.Error())
}

func (h *PublicController) scheduleMoodleSync(email, dni string) {
	if h.moodleClient == nil {
		return
	}
	go func(email, dni string) {
		ctx, cancel := context.WithTimeout(context.Background(), moodleSyncTimeout)
		defer cancel()
		if err := moodle.SyncExamUser(ctx, h.db, h.moodleClient, email, dni); err != nil {
			log.Printf("moodle sync for %s / %s failed: %v", email, dni, err)
		}
	}(email, dni)
}

func (h *PublicController) writeJSONError(w http.ResponseWriter, status int, detail string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"detail": detail})
}
