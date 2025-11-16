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
	"github.com/inscripcion-moodle/go-backend/internal/handlers"
	"github.com/inscripcion-moodle/go-backend/internal/middleware"
	"github.com/inscripcion-moodle/go-backend/internal/services/auth"
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

	cache, err := storage.NewRedis(cfg)
	if err != nil {
		return nil, err
	}

	publicHandler := handlers.NewPublicHandler(db, cache, cfg)
	registerHandler := handlers.NewRegisterHandler(cfg)
	authService, err := auth.New(cfg)
	if err != nil {
		return nil, err
	}
	adminHandler := handlers.NewAdminHandler(db, cache, authService, cfg)
	router := chi.NewRouter()
	router.Use(chimiddleware.RequestID)
	router.Use(chimiddleware.RealIP)
	router.Use(chimiddleware.Logger)
	router.Use(chimiddleware.Recoverer)
	router.Use(chimiddleware.StripSlashes)
	router.Use(cors.Handler(cors.Options{
		AllowedOrigins:   cfg.AllowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "OPTIONS", "PUT", "DELETE"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	rateLimiter := middleware.NewRateLimiter(cfg.RateLimitRequests, cfg.RateLimitWindow)
	router.Use(rateLimiter.Middleware)

	router.Post("/register", registerHandler.Register)
	router.Post("/submit-exam", publicHandler.SubmitExam)
	router.Get("/exams", publicHandler.GetExams)
	router.Get("/exams/{exam_id}/questions", publicHandler.GetQuestionStubs)
	router.Post("/check_submission", publicHandler.CheckSubmission)
	router.Route("/admin", func(r chi.Router) {
		adminHandler.RegisterRoutes(r)
	})

	httpServer := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Port),
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
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
