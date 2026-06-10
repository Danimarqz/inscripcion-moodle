package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"gorm.io/gorm"

	"github.com/inscripcion-moodle/go-backend/internal/helpers"
	"github.com/inscripcion-moodle/go-backend/internal/models"
	"github.com/inscripcion-moodle/go-backend/internal/services/moodle"
)

// sseEvent writes an SSE event and flushes immediately.
func sseEvent(w http.ResponseWriter, flusher http.Flusher, data string) {
	fmt.Fprintf(w, "data: %s\n\n", data)
	flusher.Flush()
}

// sseHeaders sets SSE headers and returns the Flusher.
// Returns (nil, false) if the ResponseWriter does not support streaming.
func sseHeaders(w http.ResponseWriter) (http.Flusher, bool) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		return nil, false
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("X-Accel-Buffering", "no")
	return flusher, true
}

func (h *AdminController) syncMoodleUsers(w http.ResponseWriter, r *http.Request) {
	if h.moodleClient == nil {
		http.Error(w, "moodle not configured", http.StatusServiceUnavailable)
		return
	}

	flusher, ok := sseHeaders(w)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	var users []models.ExamUser
	if err := h.db.Select("id", "email", "dni").Where("moodle_id IS NULL").Find(&users).Error; err != nil {
		sseEvent(w, flusher, `{"status":"error","message":"failed to load exam users"}`)
		return
	}

	if len(users) == 0 {
		sseEvent(w, flusher, `{"status":"done","checked":0,"synced":0,"failed":0}`)
		return
	}

	sseEvent(w, flusher, fmt.Sprintf(`{"status":"fetching","message":"Conectando con Moodle (%d usuarios pendientes)..."}`, len(users)))

	ctx, cancel := context.WithTimeout(r.Context(), moodleAdminSyncTimeout)
	defer cancel()

	moodleUsers, err := h.moodleClient.GetAllUsersDNI(ctx, h.cfg.MoodleExamCourseID)
	if err != nil {
		log.Printf("moodle sync-users: failed to fetch moodle users for course %d: %v", h.cfg.MoodleExamCourseID, err)
		msg, _ := json.Marshal(map[string]string{"status": "error", "message": err.Error()})
		sseEvent(w, flusher, string(msg))
		return
	}

	log.Printf("moodle sync-users: course_id=%d moodle_enrolled=%d pending_exam_users=%d",
		h.cfg.MoodleExamCourseID, len(moodleUsers), len(users))

	sseEvent(w, flusher, fmt.Sprintf(`{"status":"syncing","message":"Procesando %d usuarios..."}`, len(users)))

	checked := len(users)
	userPtrs := make([]*models.ExamUser, len(users))
	for i := range users {
		userPtrs[i] = &users[i]
	}
	synced, failed := h.syncMoodleIDsByEmail(userPtrs, moodleUsers)

	log.Printf("AUDIT: moodle sync-users finished: checked=%d synced=%d failed=%d", checked, synced, failed)

	result, _ := json.Marshal(map[string]any{
		"status":  "done",
		"checked": checked,
		"synced":  synced,
		"failed":  failed,
	})
	sseEvent(w, flusher, string(result))
}

// syncMoodleIDsByEmail assigns each user's Moodle ID by matching its email
// against the Moodle enrolment list, updating only rows that still have a NULL
// moodle_id. Returns how many were updated (synced) and how many could not be
// matched or updated (failed). Single responsibility shared by both sync flows.
func (h *AdminController) syncMoodleIDsByEmail(users []*models.ExamUser, moodleUsers []moodle.MoodleUserDNI) (synced, failed int) {
	idByEmail := make(map[string]int, len(moodleUsers))
	for _, mu := range moodleUsers {
		if e := strings.ToLower(strings.TrimSpace(mu.Email)); e != "" {
			idByEmail[e] = mu.ID
		}
	}
	for _, user := range users {
		email := strings.ToLower(strings.TrimSpace(user.Email))
		if email == "" {
			failed++
			continue
		}
		moodleID, ok := idByEmail[email]
		if !ok {
			failed++
			continue
		}
		if err := h.db.Model(&models.ExamUser{}).
			Where("id = ?", user.ID).
			Where("moodle_id IS NULL").
			Update("moodle_id", moodleID).Error; err != nil {
			log.Printf("moodle sync: assign moodle_id for user %d (%s) failed: %v", user.ID, email, err)
			failed++
			continue
		}
		synced++
	}
	return synced, failed
}

func (h *AdminController) syncOfficialResultsMoodle(w http.ResponseWriter, r *http.Request) {
	if h.moodleClient == nil {
		http.Error(w, "moodle not configured", http.StatusServiceUnavailable)
		return
	}

	examID, err := parseUintURLParam(r, "exam_id")
	if err != nil {
		http.Error(w, "invalid exam ID", http.StatusBadRequest)
		return
	}

	flusher, ok := sseHeaders(w)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	sseEvent(w, flusher, `{"status":"loading","message":"Cargando resultados sin vincular..."}`)

	var officialResults []models.ExamOfficialResult
	if err := h.db.Where("exam_id = ? AND user_id IS NULL", examID).Find(&officialResults).Error; err != nil {
		sseEvent(w, flusher, `{"status":"error","message":"failed to fetch official results"}`)
		return
	}

	if len(officialResults) == 0 {
		sseEvent(w, flusher, `{"status":"done","matched":0}`)
		return
	}

	var allUsers []models.ExamUser
	if err := h.db.Find(&allUsers).Error; err != nil {
		sseEvent(w, flusher, `{"status":"error","message":"failed to fetch exam users"}`)
		return
	}

	normalizeForMatch := func(s string) string {
		return strings.ReplaceAll(helpers.NormalizeName(s), " ", "")
	}

	sseEvent(w, flusher, fmt.Sprintf(`{"status":"pass1","message":"Pass 1: cruzando %d resultados con usuarios existentes..."}`, len(officialResults)))

	matchedCount := 0
	var remainingResults []*models.ExamOfficialResult
	// Pass-1 matched users that still lack a Moodle ID; backfilled below once the
	// Moodle user list is fetched, so they count as fully linked.
	usersNeedingMoodleID := make(map[uint]*models.ExamUser)

	// Pass 1: Match against existing ExamUsers
	for i := range officialResults {
		res := &officialResults[i]

		resFullName := normalizeForMatch(res.Nombre + res.Apellido1)
		if res.Apellido2 != nil {
			resFullName = normalizeForMatch(res.Nombre + res.Apellido1 + *res.Apellido2)
		}

		var matchedUser *models.ExamUser

		for j := range allUsers {
			user := &allUsers[j]
			resDNI := strings.ToUpper(strings.TrimSpace(res.DniMasked))
			userDNI := strings.ToUpper(strings.TrimSpace(user.DNI))

			dniMatch := false
			if len(resDNI) > 0 && len(userDNI) > 0 {
				dniMatch = helpers.MatchMaskedDNI(resDNI, userDNI)
			}

			userFullName := normalizeForMatch(user.Name + user.Surname)
			if resFullName == userFullName {
				matchedUser = user
				break
			}
			if dniMatch {
				resTokens := strings.Fields(helpers.NormalizeName(res.Nombre + " " + res.Apellido1))
				if res.Apellido2 != nil {
					resTokens = append(resTokens, strings.Fields(helpers.NormalizeName(*res.Apellido2))...)
				}
				userNameNorm := helpers.NormalizeName(user.Name + " " + user.Surname)
				nameScore := 0
				for _, tok := range resTokens {
					if len(tok) > 2 && strings.Contains(userNameNorm, tok) {
						nameScore++
					}
				}
				if nameScore >= 2 {
					matchedUser = user
					break
				}
			}
		}

		if matchedUser != nil {
			if matchedUser.MoodleID == nil {
				usersNeedingMoodleID[matchedUser.ID] = matchedUser
			}

			if err := h.db.Model(&models.ExamOfficialResult{}).Where("id = ?", res.ID).Update("user_id", matchedUser.ID).Error; err == nil {
				matchedCount++
				res.UserID = &matchedUser.ID
				log.Printf("AUDIT: syncOfficialResultsMoodle Pass1: linked Official Result ID %d to ExamUser ID %d", res.ID, matchedUser.ID)
			}
		} else {
			remainingResults = append(remainingResults, res)
		}
	}

	if len(remainingResults) == 0 && len(usersNeedingMoodleID) == 0 {
		log.Printf("AUDIT: syncOfficialResultsMoodle: exam_id=%d pass1 matched=%d (no pass2 needed)", examID, matchedCount)
		result, _ := json.Marshal(map[string]any{"status": "done", "matched": matchedCount})
		sseEvent(w, flusher, string(result))
		return
	}

	sseEvent(w, flusher, fmt.Sprintf(`{"status":"pass2","message":"Pass 1: %d coincidencias. Conectando con Moodle (%d sin vincular, %d sin Moodle ID)..."}`, matchedCount, len(remainingResults), len(usersNeedingMoodleID)))

	moodleCtx, moodleCancel := context.WithTimeout(r.Context(), moodleAdminSyncTimeout)
	defer moodleCancel()

	moodleUsers, err := h.moodleClient.GetAllUsersDNI(moodleCtx, h.cfg.MoodleExamCourseID)
	if err != nil {
		log.Printf("syncOfficialResultsMoodle: failed to fetch moodle users DNI: %v", err)
		msg, _ := json.Marshal(map[string]string{"status": "error", "message": err.Error()})
		sseEvent(w, flusher, string(msg))
		return
	}

	// Backfill Moodle IDs for Pass-1 matched users that lacked one, reusing the
	// shared email-based assignment.
	if len(usersNeedingMoodleID) > 0 {
		pending := make([]*models.ExamUser, 0, len(usersNeedingMoodleID))
		for _, u := range usersNeedingMoodleID {
			pending = append(pending, u)
		}
		if backfilled, _ := h.syncMoodleIDsByEmail(pending, moodleUsers); backfilled > 0 {
			log.Printf("AUDIT: syncOfficialResultsMoodle: exam_id=%d backfilled moodle_id for %d pass1-matched users", examID, backfilled)
		}
	}

	if len(remainingResults) == 0 {
		log.Printf("AUDIT: syncOfficialResultsMoodle: exam_id=%d pass1+backfill matched=%d (no pass2 needed)", examID, matchedCount)
		result, _ := json.Marshal(map[string]any{"status": "done", "matched": matchedCount})
		sseEvent(w, flusher, string(result))
		return
	}

	sseEvent(w, flusher, fmt.Sprintf(`{"status":"syncing","message":"Pass 2: procesando %d resultados con Moodle..."}`, len(remainingResults)))

	// Pass 2: Match remaining against Moodle DB and auto-create ExamUser
	log.Printf("syncOfficialResultsMoodle: fetched %d users from Moodle DB. Proceeding to match %d unlinked official results.", len(moodleUsers), len(remainingResults))
	for _, res := range remainingResults {
		resFullName := normalizeForMatch(res.Nombre + res.Apellido1)
		if res.Apellido2 != nil {
			resFullName = normalizeForMatch(res.Nombre + res.Apellido1 + *res.Apellido2)
		}

		resDNI := strings.ToUpper(strings.TrimSpace(res.DniMasked))

		hasNumbers := false
		for _, r := range resDNI {
			if r >= '0' && r <= '9' {
				hasNumbers = true
				break
			}
		}

		if !hasNumbers {
			log.Printf("syncOfficialResultsMoodle: skipped result ID %d due to invalid mask (original: '%s')", res.ID, res.DniMasked)
			continue
		}

		var possibleMatches []moodle.MoodleUserDNI
		for _, mu := range moodleUsers {
			muDNI := strings.ToUpper(strings.TrimSpace(mu.DNI))
			if helpers.MatchMaskedDNI(resDNI, muDNI) {
				possibleMatches = append(possibleMatches, mu)
			}
		}

		var matchedMoodleUser *moodle.MoodleUserDNI

		if len(possibleMatches) >= 1 {
			if len(possibleMatches) == 1 {
				log.Printf("syncOfficialResultsMoodle: single DNI match found for result ID %d (Mask: %s). Enforcing name match...", res.ID, resDNI)
			} else {
				log.Printf("syncOfficialResultsMoodle: multiple DNI matches (%d) found for result ID %d (Mask: %s). Attempting name match...", len(possibleMatches), res.ID, resDNI)
			}

			resTokens := strings.Fields(helpers.NormalizeName(res.Nombre + " " + res.Apellido1))
			if res.Apellido2 != nil {
				resTokens = append(resTokens, strings.Fields(helpers.NormalizeName(*res.Apellido2))...)
			}

			var bestMatch *moodle.MoodleUserDNI
			maxMatchScore := 0

			for i, pm := range possibleMatches {
				pmFullName := helpers.NormalizeName(pm.Firstname + " " + pm.Lastname)
				score := 0

				for _, token := range resTokens {
					if len(token) > 2 && strings.Contains(pmFullName, token) {
						score++
					}
				}

				if resFullName == strings.ReplaceAll(pmFullName, " ", "") {
					bestMatch = &possibleMatches[i]
					log.Printf("syncOfficialResultsMoodle: exact name match successful for result ID %d. Moodle User ID: %d", res.ID, bestMatch.ID)
					break
				}

				if score > maxMatchScore && score >= 2 {
					maxMatchScore = score
					bestMatch = &possibleMatches[i]
				}
			}

			if bestMatch != nil {
				matchedMoodleUser = bestMatch
				log.Printf("syncOfficialResultsMoodle: name match successful for result ID %d (Score: %d). Moodle User ID: %d", res.ID, maxMatchScore, matchedMoodleUser.ID)
			}

			if matchedMoodleUser == nil {
				log.Printf("syncOfficialResultsMoodle: name match failed for result ID %d (Mask: %s). Could not validate user.", res.ID, resDNI)
			}
		} else {
			log.Printf("syncOfficialResultsMoodle: zero DNI matches found for result ID %d (Mask: %s).", res.ID, resDNI)
		}

		if matchedMoodleUser != nil {
			newUser := models.ExamUser{
				Email:            strings.ToLower(strings.TrimSpace(matchedMoodleUser.Email)),
				Name:             helpers.NormalizeName(matchedMoodleUser.Firstname),
				Surname:          helpers.NormalizeName(matchedMoodleUser.Lastname),
				DNI:              matchedMoodleUser.DNI,
				MoodleID:         &matchedMoodleUser.ID,
				AcceptsMarketing: false,
			}

			txErr := h.db.Transaction(func(tx *gorm.DB) error {
				var existingUser models.ExamUser
				if err := tx.Where("email = ?", newUser.Email).First(&existingUser).Error; err == nil {
					log.Printf("syncOfficialResultsMoodle: auto-create skipped for result ID %d because email %s already exists. Updating its MoodleID instead.", res.ID, newUser.Email)
					if err := tx.Model(&existingUser).Update("moodle_id", matchedMoodleUser.ID).Error; err != nil {
						return err
					}
					return tx.Model(&models.ExamOfficialResult{}).Where("id = ?", res.ID).Update("user_id", existingUser.ID).Error
				}
				if err := tx.Create(&newUser).Error; err != nil {
					return err
				}
				return tx.Model(&models.ExamOfficialResult{}).Where("id = ?", res.ID).Update("user_id", newUser.ID).Error
			})
			if txErr == nil {
				matchedCount++
				log.Printf("AUDIT: syncOfficialResultsMoodle Pass2: linked Official Result ID %d to ExamUser (email: %s)", res.ID, newUser.Email)
				if newUser.ID != 0 {
					log.Printf("syncOfficialResultsMoodle: successfully auto-created ExamUser ID %d for Official Result ID %d", newUser.ID, res.ID)
				}
			} else {
				log.Printf("syncOfficialResultsMoodle: transaction failed for Official Result ID %d: %v", res.ID, txErr)
			}
		}
	}

	log.Printf("AUDIT: syncOfficialResultsMoodle: exam_id=%d pass1+pass2 matched=%d", examID, matchedCount)
	result, _ := json.Marshal(map[string]any{"status": "done", "matched": matchedCount})
	sseEvent(w, flusher, string(result))
}
