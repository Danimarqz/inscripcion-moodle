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
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"github.com/inscripcion-moodle/go-backend/internal/config"
	"github.com/inscripcion-moodle/go-backend/internal/models"
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
	cache        *redis.Client
	cacheTTL     time.Duration
	moodleClient *moodle.Client
}

func NewPublicController(db *gorm.DB, cache *redis.Client, cfg *config.Config) *PublicController {
	return &PublicController{
		db:           db,
		cache:        cache,
		cacheTTL:     cfg.PublicCacheTTL,
		moodleClient: moodle.New(cfg),
	}
}

func (h *PublicController) GetExams(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if cached, ok := h.readCache(ctx, examsCacheKey); ok {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(cached)
		return
	}

	var exams []models.Exam
	if err := h.db.Where("is_active = ?", true).Find(&exams).Error; err != nil {
		http.Error(w, "failed to load exams", http.StatusInternalServerError)
		return
	}

	payload, err := h.writeJSON(w, exams)
	if err != nil {
		return
	}
	h.setCache(ctx, examsCacheKey, payload, 0)
}

func (h *PublicController) SubmitExam(w http.ResponseWriter, r *http.Request) {
	var req examservice.SubmitExamRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	payload, err := examservice.ProcessExamSubmission(h.db, req)
	if err != nil {
		h.handleError(w, err)
		return
	}

	h.invalidateCheckCacheForExam(req.ExamID)
	h.scheduleMoodleSync(req.Email, req.DNI)

	_, _ = h.writeJSON(w, payload)
}

func (h *PublicController) GetQuestionStubs(w http.ResponseWriter, r *http.Request) {
	examIDParam := chi.URLParam(r, "exam_id")
	examID, err := strconv.ParseUint(examIDParam, 10, 64)
	if err != nil {
		http.Error(w, "invalid exam id", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	key := h.questionsCacheKey(uint(examID))
	if cached, ok := h.readCache(ctx, key); ok {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(cached)
		return
	}

	var exam models.Exam
	if err := h.db.First(&exam, uint(examID)).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			http.Error(w, "exam not found", http.StatusNotFound)
			return
		}
		http.Error(w, "failed to load exam questions", http.StatusInternalServerError)
		return
	}

	var questions []models.Question
	if err := h.db.Where("exam_id = ?", uint(examID)).Find(&questions).Error; err != nil {
		http.Error(w, "failed to load questions", http.StatusInternalServerError)
		return
	}

	sort.SliceStable(questions, func(i, j int) bool {
		if questions[i].Name == questions[j].Name {
			return questions[i].ID < questions[j].ID
		}
		return questions[i].Name < questions[j].Name
	})

	stubs := make([]QuestionStub, 0, len(questions))
	for _, question := range questions {
		stubs = append(stubs, QuestionStub{
			ID:          question.ID,
			Name:        question.Name,
			IsActive:    question.IsActive,
			IsCancelled: question.IsCancelled,
		})
	}

	payload, err := h.writeJSON(w, stubs)
	if err != nil {
		return
	}
	h.setCache(ctx, key, payload, 0)
}

func (h *PublicController) CheckSubmission(w http.ResponseWriter, r *http.Request) {
	var req examservice.SubmissionCheckRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	cacheKey := h.submissionCacheKey(req.ExamID, req.Email, req.DNI)
	if cached, ok := h.readCache(ctx, cacheKey); ok {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(cached)
		return
	}

	payload, err := examservice.BuildSubmissionCheckResponse(h.db, req)
	if err != nil {
		h.handleError(w, err)
		return
	}

	data, err := h.writeJSON(w, payload)
	if err != nil {
		return
	}
	h.setCache(ctx, cacheKey, data, h.cacheTTL)
}

func (h *PublicController) writeJSON(w http.ResponseWriter, data any) ([]byte, error) {
	payload, err := json.Marshal(data)
	if err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
		return nil, err
	}
	w.Header().Set("Content-Type", "application/json")
	if _, err := w.Write(payload); err != nil {
		return nil, err
	}
	return payload, nil
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

func (h *PublicController) readCache(ctx context.Context, key string) ([]byte, bool) {
	if h.cache == nil {
		return nil, false
	}
	payload, err := h.cache.Get(ctx, key).Bytes()
	if err == nil {
		return payload, true
	}
	if err != redis.Nil {
		_ = err
	}
	return nil, false
}

func (h *PublicController) setCache(ctx context.Context, key string, payload []byte, ttl time.Duration) {
	if h.cache == nil || ttl < 0 {
		return
	}
	_ = h.cache.Set(ctx, key, payload, ttl).Err()
}

func (h *PublicController) invalidateCheckCacheForExam(examID uint) {
	if h.cache == nil {
		return
	}
	ctx := context.Background()
	pattern := fmt.Sprintf("%s:%d:*", submissionCachePrefix, examID)
	iter := h.cache.Scan(ctx, 0, pattern, 250).Iterator()
	for iter.Next(ctx) {
		_ = h.cache.Del(ctx, iter.Val()).Err()
	}
}

func (h *PublicController) handleError(w http.ResponseWriter, err error) {
	status := http.StatusBadRequest
	switch {
	case errors.Is(err, examservice.ErrSubmissionNotFound), errors.Is(err, examservice.ErrExamNotFound):
		status = http.StatusNotFound
	case errors.Is(err, examservice.ErrExamNotActive), errors.Is(err, examservice.ErrResultsNotViewable):
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
