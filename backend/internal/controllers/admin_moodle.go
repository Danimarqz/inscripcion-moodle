package controllers

import (
	"context"
	"log"
	"net/http"
	"strings"

	"gorm.io/gorm"

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

	log.Printf("AUDIT: moodle sync finished: checked=%d synced=%d failed=%d", checked, synced, failed)
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

	// Fetch unlinked official results for this exam
	var officialResults []models.ExamOfficialResult
	if err := h.db.Where("exam_id = ? AND user_id IS NULL", examID).Find(&officialResults).Error; err != nil {
		http.Error(w, "failed to fetch official results", http.StatusInternalServerError)
		return
	}

	if len(officialResults) == 0 {
		writeJSON(w, map[string]any{
			"status":  "success",
			"matched": 0,
			"synced":  0,
		})
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
	var remainingResults []*models.ExamOfficialResult

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
				// Strict positional matching from the right side (ignoring final letter if missing in mask)
				dniMatch = matchMaskedDNI(resDNI, userDNI)
			}

			if !dniMatch {
				continue
			}

			userFullName := normalizeForMatch(user.Name + user.Surname)
			if resFullName == userFullName {
				matchedUser = user
				break
			}
		}

		if matchedUser != nil {
			if err := h.db.Model(&models.ExamOfficialResult{}).Where("id = ?", res.ID).Update("user_id", matchedUser.ID).Error; err == nil {
				matchedCount++
				res.UserID = &matchedUser.ID
				log.Printf("AUDIT: syncOfficialResultsMoodle Pass1: linked Official Result ID %d to ExamUser ID %d", res.ID, matchedUser.ID)
			}
		} else {
			remainingResults = append(remainingResults, res)
		}
	}

	// Pass 2: Match remaining against Moodle DB and auto-create ExamUser
	if len(remainingResults) > 0 {
		moodleCtx, moodleCancel := context.WithTimeout(context.Background(), moodleAdminSyncTimeout)
		defer moodleCancel()
		moodleUsers, err := h.moodleClient.GetAllUsersDNI(moodleCtx)
		if err != nil {
			log.Printf("syncOfficialResultsMoodle: failed to fetch moodle users DNI: %v", err)
		} else {
			log.Printf("syncOfficialResultsMoodle: fetched %d users from Moodle DB. Proceeding to match %d unlinked official results.", len(moodleUsers), len(remainingResults))
			for _, res := range remainingResults {
				resFullName := normalizeForMatch(res.Nombre + res.Apellido1)
				if res.Apellido2 != nil {
					resFullName = normalizeForMatch(res.Nombre + res.Apellido1 + *res.Apellido2)
				}

				resDNI := strings.ToUpper(strings.TrimSpace(res.DniMasked))

				// Ensure at least some numbers exist
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
					if matchMaskedDNI(resDNI, muDNI) {
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

					// Check name and surname (Permissive token matching)
					// We split the official result's name and surnames into words
					resTokens := strings.Fields(helpers.NormalizeName(res.Nombre + " " + res.Apellido1))
					if res.Apellido2 != nil {
						resTokens = append(resTokens, strings.Fields(helpers.NormalizeName(*res.Apellido2))...)
					}

					var bestMatch *moodle.MoodleUserDNI
					maxMatchScore := 0

					for i, pm := range possibleMatches {
						pmFullName := helpers.NormalizeName(pm.Firstname + " " + pm.Lastname)
						score := 0

						// Count how many tokens from the official result exist in the Moodle name
						for _, token := range resTokens {
							if len(token) > 2 && strings.Contains(pmFullName, token) {
								score++
							}
						}

						// Exact match is still instantly chosen
						if resFullName == strings.ReplaceAll(pmFullName, " ", "") {
							bestMatch = &possibleMatches[i]
							log.Printf("syncOfficialResultsMoodle: exact name match successful for result ID %d. Moodle User ID: %d", res.ID, bestMatch.ID)
							break
						}

						// Keep track of the highest score (needs at least 2 matching words to be considered safe)
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
					// Create ExamUser since it didn't exist
					newUser := models.ExamUser{
						Email:            strings.ToLower(strings.TrimSpace(matchedMoodleUser.Email)),
						Name:             helpers.NormalizeName(matchedMoodleUser.Firstname),
						Surname:          helpers.NormalizeName(matchedMoodleUser.Lastname),
						DNI:              matchedMoodleUser.DNI,
						MoodleID:         &matchedMoodleUser.ID,
						AcceptsMarketing: false, // Default
					}

					txErr := h.db.Transaction(func(tx *gorm.DB) error {
						// We check if email somehow exists just in case
						var existingUser models.ExamUser
						if err := tx.Where("email = ?", newUser.Email).First(&existingUser).Error; err == nil {
							// Found existing by email that was missed due to DNI mismatch? Just update MoodleID
							log.Printf("syncOfficialResultsMoodle: auto-create skipped for result ID %d because email %s already exists. Updating its MoodleID instead.", res.ID, newUser.Email)
							if err := tx.Model(&existingUser).Update("moodle_id", matchedMoodleUser.ID).Error; err != nil {
								return err
							}
							return tx.Model(&models.ExamOfficialResult{}).Where("id = ?", res.ID).Update("user_id", existingUser.ID).Error
						}
						// Create new
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
		}
	}

	log.Printf("AUDIT: syncOfficialResultsMoodle: exam_id=%d pass1+pass2 matched=%d", examID, matchedCount)
	writeJSON(w, map[string]any{
		"status":  "success",
		"matched": matchedCount,
		"synced":  0,
	})
}

// matchMaskedDNI compares a masked DNI (e.g. "***7090**") with a full DNI (e.g. "12370908Z")
// It assumes the full DNI is always complete (with letters).
func matchMaskedDNI(masked, full string) bool {
	masked = strings.ToUpper(strings.TrimSpace(masked))
	full = strings.ToUpper(strings.TrimSpace(full))

	if masked == full {
		return true
	}

	// If they don't have the same number of characters, the comparison is invalid
	if len(masked) != len(full) {
		return false
	}

	for i := 0; i < len(masked); i++ {
		mChar := masked[i]
		fChar := full[i]

		// Any character that is not a letter or a number is treated as a wildcard mask
		isAlphaNum := (mChar >= '0' && mChar <= '9') || (mChar >= 'A' && mChar <= 'Z')
		if !isAlphaNum {
			continue
		}

		if mChar != fChar {
			return false
		}
	}

	return true
}
