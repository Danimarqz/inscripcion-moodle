package moodle

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/inscripcion-moodle/go-backend/internal/models"
	"gorm.io/gorm"
)

var (
	ErrUserNotManual = errors.New("moodle user is not manually authenticated")
)

func SyncExamUser(ctx context.Context, db *gorm.DB, client *Client, email, dni string) error {
	if client == nil {
		return ErrNotConfigured
	}
	if db == nil {
		return errors.New("database is required")
	}
	normalizedDNI := normalizeDNI(dni)
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

	var moodleID int
	for _, candidate := range users {
		if strings.EqualFold(candidate.Auth, "manual") {
			moodleID = candidate.ID
			break
		}
	}
	if moodleID == 0 {
		return ErrUserNotManual
	}

	return db.Model(&user).Where("moodle_id IS NULL").Update("moodle_id", moodleID).Error
}

func normalizeDNI(value string) string {
	return strings.ToUpper(strings.TrimSpace(value))
}
