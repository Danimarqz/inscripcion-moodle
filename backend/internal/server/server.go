package server

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"github.com/inscripcion-moodle/go-backend/internal/config"
	controllers "github.com/inscripcion-moodle/go-backend/internal/controllers"
	"github.com/inscripcion-moodle/go-backend/internal/middleware"
	"github.com/inscripcion-moodle/go-backend/internal/repository"
	"github.com/inscripcion-moodle/go-backend/internal/services/auth"
	"github.com/inscripcion-moodle/go-backend/internal/services/email"
	examservice "github.com/inscripcion-moodle/go-backend/internal/services/exam"
	"github.com/inscripcion-moodle/go-backend/internal/services/logs"
	"github.com/inscripcion-moodle/go-backend/internal/services/moodle"
	"github.com/inscripcion-moodle/go-backend/internal/storage"
)

type Server struct {
	httpServer *http.Server
	db         *gorm.DB
	cache      *redis.Client
}

func New(cfg *config.Config) (*Server, error) {
	db, err := storage.NewMariaDB(cfg)
	if err != nil {
		return nil, err
	}

	// AutoMigrate removed as per user request

	cache, err := storage.NewRedis(cfg)
	if err != nil {
		return nil, err
	}

	examRepo := repository.NewExamRepository()
	submissionRepo := repository.NewSubmissionRepository()
	examService := examservice.NewService(db, examRepo, submissionRepo, cfg.SMTPFrom)

	// Initialize background worker pools
	email.InitWorkerPool(3)
	moodle.InitSyncWorkerPool(5)
	logs.SetIPHashSalt(cfg.IPHashSalt)
	logs.SetSummaryDir(cfg.LogsSummaryDir)

	publicController := controllers.NewPublicController(db, cache, cfg, examService)
	registerController := controllers.NewRegisterController(cfg)
	authService, err := auth.New(cfg)
	if err != nil {
		return nil, err
	}
	adminController := controllers.NewAdminController(db, cache, authService, cfg)
	logsController := controllers.NewLogsController(cache, cfg)
	router := chi.NewRouter()
	router.Use(chimiddleware.RequestID)
	router.Use(chimiddleware.RealIP)
	router.Use(chimiddleware.Logger)
	router.Use(chimiddleware.Recoverer)
	router.Use(chimiddleware.StripSlashes)
	router.Use(cors.Handler(cors.Options{
		AllowedOrigins:   cfg.AllowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "OPTIONS", "PUT", "DELETE", "PATCH"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	rateLimiter := middleware.NewRateLimiter(cfg.RateLimitRequests, cfg.RateLimitWindow)
	router.Use(rateLimiter.Middleware)

	router.Post("/register", registerController.Register)
	router.Post("/submit-exam", publicController.SubmitExam)
	router.Post("/check-official-result", publicController.CheckOfficialResultMatch)
	router.Get("/exams", publicController.GetExams)
	router.Get("/exams/slug/{slug}", publicController.GetExamBySlug)
	router.Get("/exams/{exam_id}/questions", publicController.GetQuestionStubs)
	router.Post("/check_submission", publicController.CheckSubmission)
	router.Post("/update-merits", publicController.UpdateMerits)
	router.Get("/logs/stats", logsController.GetStats)
	router.Get("/logs/sites", logsController.GetSites)
	router.Get("/api/questions/{question_id}/feedback.m3u8", publicController.GetFeedbackPlaylist)
	router.Route("/admin", func(r chi.Router) {
		adminController.RegisterRoutes(r)
	})

	// WriteTimeout must accommodate the slowest endpoint; /logs/stats can
	// take dozens of seconds when parsing many gzipped files without cache.
	httpServer := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Port),
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 240 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return &Server{
		httpServer: httpServer,
		db:         db,
		cache:      cache,
	}, nil
}

func (s *Server) Start(ctx context.Context) error {
	errCh := make(chan error, 1)
	go func() {
		log.Printf("server listening on %s", s.httpServer.Addr)
		errCh <- s.httpServer.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		if err == http.ErrServerClosed {
			return nil
		}
		return err
	case <-ctx.Done():
		return s.shutdown()
	}
}

func (s *Server) shutdown() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := s.httpServer.Shutdown(ctx); err != nil {
		return err
	}

	sqlDB, err := s.db.DB()
	if err == nil {
		_ = sqlDB.Close()
	}

	if s.cache != nil {
		_ = s.cache.Close()
	}

	return nil
}
