package register

import (
	"context"
	"log"
	"net/http"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/inscripcion-moodle/go-backend/internal/config"
)

type Service struct {
	cfg    *config.Config
	client *http.Client
}

func New(cfg *config.Config) *Service {
	return &Service{
		cfg: cfg,
		client: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

func (s *Service) Register(ctx context.Context, data Data) (*Result, error) {
	var (
		pdfBytes     []byte
		moodleFailed bool
	)
	// aumentar tiempo de ejecución para llamdas a moodle
	ctx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()
	group, ctx := errgroup.WithContext(ctx)

	group.Go(func() error {
		var err error
		pdfBytes, err = generatePDF(data)
		return err
	})

	group.Go(func() error {
		conflict, err := s.createMoodleUser(ctx, data)
		if err != nil {
			log.Printf("register: moodle error: %v", err)
			moodleFailed = true
			return nil
		}
		if conflict {
			moodleFailed = true
		}
		return nil
	})

	group.Go(func() error {
		return postRegistrationToGSheet(ctx, s.client, s.cfg.GSheetAPI, data)
	})

	if err := group.Wait(); err != nil {
		return nil, err
	}

	if err := sendEmails(s.cfg, data.Email, pdfBytes, data.Name, data.Surname, moodleFailed); err != nil {
		return nil, err
	}

	message := "Inscripción completada correctamente"
	if moodleFailed {
		message = "Tu inscripción se gestionará lo antes posible"
	}

	return &Result{Message: message}, nil
}
