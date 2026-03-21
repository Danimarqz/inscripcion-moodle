package moodle

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/inscripcion-moodle/go-backend/internal/models"
	"gorm.io/gorm"
)

type SyncJob struct {
	Email  string
	DNI    string
	DB     *gorm.DB
	Client *Client
}

var syncQueue chan SyncJob

func InitSyncWorkerPool(workers int) {
	syncQueue = make(chan SyncJob, 1000)
	for range workers {
		go func() {
			for job := range syncQueue {
				func() {
					ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
					defer cancel()
					if err := SyncExamUser(ctx, job.DB, job.Client, job.Email, job.DNI); err != nil {
						log.Printf("moodle sync worker error for %s / %s: %v", job.Email, job.DNI, err)
					}
				}()
			}
		}()
	}
}

func EnqueueSyncUser(db *gorm.DB, client *Client, email, dni string) error {
	if syncQueue == nil {
		return errors.New("moodle sync worker pool not initialized")
	}

	select {
	case syncQueue <- SyncJob{
		Email:  email,
		DNI:    dni,
		DB:     db,
		Client: client,
	}:
		return nil
	default:
		return errors.New("moodle sync queue is full")
	}
}

var (
	ErrUserNotEnrolled = errors.New("moodle user is not enrolled in the required course")
)

func SyncExamUser(ctx context.Context, db *gorm.DB, client *Client, email, dni string) error {
	if client == nil {
		return ErrNotConfigured
	}
	if db == nil {
		return errors.New("database is required")
	}
	normalizedDNI := strings.ToUpper(strings.TrimSpace(dni))
	if normalizedDNI == "" {
		return fmt.Errorf("invalid dni")
	}

	var user models.ExamUser
	if err := db.Where("dni = ?", normalizedDNI).First(&user).Error; err != nil {
		return err
	}
	if user.MoodleID != nil {
		return nil
	}

	emailValue := strings.ToLower(strings.TrimSpace(email))
	if emailValue == "" {
		return fmt.Errorf("invalid email for dni %s", normalizedDNI)
	}

	users, err := client.FindUsersByField(ctx, "email", emailValue)
	if err != nil {
		return err
	}

	var (
		moodleID int
	)
	for _, candidate := range users {
		enrolled, err := client.IsUserEnrolledInCourse(ctx, client.moodleExamCourseID, candidate.ID)
		if err != nil {
			return err
		}
		if !enrolled {
			continue
		}
		moodleID = candidate.ID
		break
	}
	if moodleID == 0 {
		return ErrUserNotEnrolled
	}

	return db.Model(&user).Where("moodle_id IS NULL").Update("moodle_id", moodleID).Error
}

