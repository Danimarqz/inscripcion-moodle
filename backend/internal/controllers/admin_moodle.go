package controllers

import (
	"context"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/inscripcion-moodle/go-backend/internal/helpers"
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

	enrolledUsers, err := h.moodleClient.GetEnrolledUsers(ctx, h.cfg.MoodleExamCourseID, nil)
	if err != nil {
		log.Printf("failed to fetch enrolled users for course %d: %v", h.cfg.MoodleExamCourseID, err)
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
			log.Printf("moodle sync for %s (%s) skipped: not enrolled in course %d", user.Email, user.DNI, h.cfg.MoodleExamCourseID)
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

func (h *AdminController) syncOfficialResultsMoodle(w http.ResponseWriter, r *http.Request) {
	if h.moodleClient == nil {
		http.Error(w, "moodle not configured", http.StatusServiceUnavailable)
		return
	}

	examIDStr := chi.URLParam(r, "exam_id")
	examID, err := strconv.ParseUint(examIDStr, 10, 32)
	if err != nil {
		http.Error(w, "invalid exam ID", http.StatusBadRequest)
		return
	}

	// Fetch unlinked official results for this exam
	var officialResults []models.ExamOfficialResult
	if err := h.db.Where("exam_id = ? AND user_id IS NULL", examID).Find(&officialResults).Error; err != nil {
		http.Error(w, "failed to fetch official results", http.StatusInternalServerError)
		return
	}

	// Fetch all exam users
	var allUsers []models.ExamUser
	if err := h.db.Find(&allUsers).Error; err != nil {
		http.Error(w, "failed to fetch exam users", http.StatusInternalServerError)
		return
	}

	// Helper to normalize strings for comparison by removing spaces too
	normalizeForMatch := func(s string) string {
		return strings.ReplaceAll(helpers.NormalizeName(s), " ", "")
	}

	matchedCount := 0
	for i := range officialResults {
		res := &officialResults[i]

		resFullName := normalizeForMatch(res.Nombre + res.Apellido1)
		if res.Apellido2 != nil {
			resFullName = normalizeForMatch(res.Nombre + res.Apellido1 + *res.Apellido2)
		}

		var matchedUserID *uint

		for _, user := range allUsers {
			// DNI Match
			resDNI := strings.ToUpper(strings.TrimSpace(res.DniMasked))
			userDNI := strings.ToUpper(strings.TrimSpace(user.DNI))

			dniMatch := false
			if len(resDNI) > 0 && len(userDNI) > 0 {
				var cleanResDNI strings.Builder
				for _, r := range resDNI {
					// Keep only numbers and letters
					if (r >= '0' && r <= '9') || (r >= 'A' && r <= 'Z') {
						cleanResDNI.WriteRune(r)
					}
				}

				if cleanResDNI.Len() > 0 && strings.Contains(userDNI, cleanResDNI.String()) {
					dniMatch = true
				}
			}

			if !dniMatch {
				continue
			}

			// Name match
			userFullName := normalizeForMatch(user.Name + user.Surname)
			if resFullName == userFullName {
				matchedUserID = &user.ID
				break
			}
		}

		if matchedUserID != nil {
			// Update the result
			if err := h.db.Model(&models.ExamOfficialResult{}).Where("id = ?", res.ID).Update("user_id", *matchedUserID).Error; err == nil {
				matchedCount++
			}
		}
	}

	h.triggerSyncByExamID(w, uint(examID), matchedCount)
}

func (h *AdminController) triggerSyncByExamID(w http.ResponseWriter, examID uint, matchedCount int) {
	// Now fetch all users associated with this exam's official results that don't have a moodle_id
	var usersToSync []models.ExamUser
	query := `
		SELECT eu.* FROM exam_user eu
		JOIN exam_official_result eor ON eu.id = eor.user_id
		WHERE eor.exam_id = ? AND eu.moodle_id IS NULL
		GROUP BY eu.id
	`
	if err := h.db.Raw(query, examID).Scan(&usersToSync).Error; err != nil {
		log.Printf("syncOfficialResultsMoodle: failed to fetch users to sync: %v", err)
	}

	syncedCount := len(usersToSync)
	if syncedCount > 0 {
		go h.syncMoodleUsersAsync(usersToSync)
	}

	writeJSON(w, map[string]any{
		"status":  "success",
		"matched": matchedCount,
		"synced":  syncedCount,
	})
}
