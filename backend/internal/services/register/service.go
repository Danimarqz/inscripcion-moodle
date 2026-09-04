package register

import (
	"context"
	"log"
	"net/http"
	"strings"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/inscripcion-moodle/go-backend/internal/config"
	"github.com/inscripcion-moodle/go-backend/internal/services/moodle"
)

type Service struct {
	cfg          *config.Config
	client       *http.Client
	moodleClient *moodle.Client
}

func New(cfg *config.Config) *Service {
	return &Service{
		cfg: cfg,
		client: &http.Client{
			// Apps Script redirige a script.googleusercontent.com y pasa de 15s con facilidad;
			// el ctx de Register (1 min) sigue siendo el límite real.
			Timeout: 45 * time.Second,
		},
		moodleClient: moodle.New(cfg),
	}
}

func (s *Service) Register(ctx context.Context, data Data) (*Result, error) {
	var (
		pdfBytes       []byte
		moodleFailed   bool
		gsheetConflict bool
		gsheetFailed   bool
		moodleUsername string
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
		conflict, un, err := s.createMoodleUser(ctx, data)
		if err != nil {
			log.Printf("register: moodle error: %v", err)
			moodleFailed = true
			return nil
		}
		moodleUsername = un
		if conflict {
			moodleFailed = true
		}
		return nil
	})

	group.Go(func() error {
		if err := postRegistrationToGSheet(ctx, s.client, s.cfg.GSheetAPI, data); err != nil {
			if strings.Contains(strings.ToLower(err.Error()), "dni ya registrado") {
				log.Printf("register: gsheet conflict (DNI ya registrado): %v", err)
				gsheetConflict = true
				return nil
			}
			// No abortamos: Moodle ya puede haber creado al usuario y el reintento chocaría con
			// el duplicado. El admin recibe el PDF y el aviso para meter la fila a mano.
			log.Printf("register: gsheet error (alta NO escrita en la hoja) dni=%s email=%s: %v", data.DNI, data.Email, err)
			gsheetFailed = true
			return nil
		}
		return nil
	})

	if err := group.Wait(); err != nil {
		return nil, err
	}

	if err := sendEmails(s.cfg, data.Email, pdfBytes, data.Name, data.Surname, moodleFailed, moodleUsername, data.DNI, gsheetConflict, gsheetFailed); err != nil {
		log.Printf("register: email send error: %v", err)
	}

	alreadyRegistered := moodleFailed || gsheetConflict

	message := "Inscripción completada correctamente"
	if alreadyRegistered {
		message = "Ya estabas registrado en la plataforma, envía un correo lo antes posible a info@opositatcae.es"
	}

	return &Result{Message: message}, nil
}
