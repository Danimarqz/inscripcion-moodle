package controllers

import (
	"context"
	"log"
	"net/http"
	"strings"

	"github.com/inscripcion-moodle/go-backend/internal/models"
	"github.com/inscripcion-moodle/go-backend/internal/services/moodle"
)

func (h *AdminController) syncMoodleUsers(w http.ResponseWriter, r *http.Request) {
	if h.moodleClient == nil {
		http.Error(w, "moodle not configured", http.StatusServiceUnavailable)
		return
	}

	var users []models.ExamUser
	if err := h.db.Select("id", "email", "dni").Where("moodle_id IS NULL").Find(&users).Error; err != nil {
		http.Error(w, "failed to load exam users", http.StatusInternalServerError)
		return
	}

	if len(users) == 0 {
		writeJSON(w, map[string]string{"status": "no users to sync"})
		return
	}

	go h.syncMoodleUsersAsync(append([]models.ExamUser(nil), users...))

	writeJSON(w, map[string]string{"status": "sync started"})
}

func (h *AdminController) syncMoodleUsersAsync(users []models.ExamUser) {
	ctx, cancel := context.WithTimeout(context.Background(), moodleAdminSyncTimeout)
	defer cancel()

	enrolledUsers, err := h.moodleClient.GetEnrolledUsers(ctx, moodle.MoodleExamCourseID, nil)
	if err != nil {
		log.Printf("failed to fetch enrolled users for course %d: %v", moodle.MoodleExamCourseID, err)
		return
	}

	enrolledByEmail := make(map[string]moodle.EnrolledUser, len(enrolledUsers))
	for _, enrolled := range enrolledUsers {
		email := strings.ToLower(strings.TrimSpace(enrolled.Email))
		if email == "" {
			continue
		}
		enrolledByEmail[email] = enrolled
	}

	checked := len(users)
	synced := 0
	failed := 0

	for _, user := range users {
		email := strings.ToLower(strings.TrimSpace(user.Email))
		if email == "" {
			log.Printf("moodle sync for %s (%s) skipped: missing email", user.Email, user.DNI)
			failed++
			continue
		}

		enrolled, ok := enrolledByEmail[email]
		if !ok {
			log.Printf("moodle sync for %s (%s) skipped: not enrolled in course %d", user.Email, user.DNI, moodle.MoodleExamCourseID)
			failed++
			continue
		}

		if err := h.db.Model(&models.ExamUser{}).
			Where("id = ?", user.ID).
			Where("moodle_id IS NULL").
			Update("moodle_id", enrolled.ID).Error; err != nil {
			log.Printf("moodle sync for %s (%s) failed: %v", user.Email, user.DNI, err)
			failed++
			continue
		}
		synced++
	}

	log.Printf("moodle sync finished: checked=%d synced=%d failed=%d", checked, synced, failed)
}
