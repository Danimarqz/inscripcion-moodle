package controllers

import (
	"context"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"github.com/inscripcion-moodle/go-backend/internal/config"
	"github.com/inscripcion-moodle/go-backend/internal/services/admin"
	"github.com/inscripcion-moodle/go-backend/internal/services/auth"
	"github.com/inscripcion-moodle/go-backend/internal/services/moodle"
	pdfimport "github.com/inscripcion-moodle/go-backend/internal/services/pdfimport"
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
