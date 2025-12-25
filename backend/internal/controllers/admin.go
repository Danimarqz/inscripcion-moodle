package controllers

import (
	"context"
	"io"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"github.com/inscripcion-moodle/go-backend/internal/cache"
	"github.com/inscripcion-moodle/go-backend/internal/config"
	"github.com/inscripcion-moodle/go-backend/internal/models"
	"github.com/inscripcion-moodle/go-backend/internal/repository"
	"github.com/inscripcion-moodle/go-backend/internal/services/admin"
	"github.com/inscripcion-moodle/go-backend/internal/services/auth"
	excelimport "github.com/inscripcion-moodle/go-backend/internal/services/excelimport"
	"github.com/inscripcion-moodle/go-backend/internal/services/moodle"
)

type contextKey string

const (
	adminContextKey         contextKey = "admin-user"
	defaultSubmissionsLimit            = 100
	maxSubmissionsLimit                = 500
	moodleAdminSyncTimeout             = 20 * time.Second
)

type AdminController struct {
	db           *gorm.DB
	cache        *cache.Cache
	auth         *auth.Service
	service      *admin.Service
	excelImport    excelImportService
	moodleClient *moodle.Client
	cfg          *config.Config
}

type excelImportService interface {
	ImportOfficialResultsExcel(context.Context, uint, io.Reader, bool) (*models.ExcelImportResult, error)
}

type adminRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type tokenResponse struct {
	AccessToken string `json:"access_token"`
}

type genericError struct {
	Message string `json:"message"`
}


func NewAdminController(db *gorm.DB, cacheClient *redis.Client, authService *auth.Service, cfg *config.Config) *AdminController {
	examRepo := repository.NewExamRepository()
	subRepo := repository.NewSubmissionRepository()
	officialRepo := repository.NewOfficialResultRepository()
	
	return &AdminController{
		db:           db,
		cache:        cache.New(cacheClient),
		auth:         authService,
		service:      admin.New(db, examRepo, subRepo, officialRepo),
		excelImport:  excelimport.New(db),
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
