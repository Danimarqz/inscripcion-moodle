package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"

	"github.com/inscripcion-moodle/go-backend/internal/config"
	"github.com/inscripcion-moodle/go-backend/internal/server"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	srv, err := server.New(ctx, cfg)
	if err != nil {
		log.Fatalf("initialize server: %v", err)
	}

	if err := srv.Start(ctx); err != nil {
		log.Fatalf("server error: %v", err)
	}

	if err := context.Cause(ctx); err != nil {
		log.Printf("shutdown cause: %v", err)
	}

	log.Println("server stopped")
}
