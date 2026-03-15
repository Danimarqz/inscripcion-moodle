package admin

import (
	"context"
	"database/sql"
	"fmt"
	"maps"
	"strings"

	"github.com/inscripcion-moodle/go-backend/internal/models"
)

func (s *Service) DeleteSubmission(submissionID uint) (examID uint, err error) {
	submission, err := s.submissionRepo.FindByID(context.Background(), s.db, submissionID)
	if err != nil {
		return 0, err
	}
	if err := s.submissionRepo.Delete(context.Background(), s.db, submissionID); err != nil {
		return 0, err
	}
	return submission.ExamID, nil
}

func (s *Service) GetSubmission(submissionID uint) (*models.UserExamSubmission, error) {
	return s.submissionRepo.FindByID(context.Background(), s.db, submissionID)
}

func (s *Service) UpdateSubmission(submissionID uint, req SubmissionUpdateRequest) (*models.UserExamSubmission, error) {
	submission, err := s.submissionRepo.FindByID(context.Background(), s.db, submissionID)
	if err != nil {
		return nil, err
	}

	if err := s.updateUserFromSubmission(submission.User, req); err != nil {
		return nil, err
	}

	if err := s.updateAnswersFromSubmission(submission, req); err != nil {
		return nil, err
	}

	if err := s.submissionRepo.Update(context.Background(), s.db, submission); err != nil {
		return nil, err
	}

	go s.recalculateScoresForSubmissionAsync(submission.ExamID, submission.ID)

	return s.submissionRepo.FindByID(context.Background(), s.db, submissionID)
}

// recalculateScoresForSubmissionAsync runs the score and percentile recalculation in a separate
// transaction and context, making it safe to run in a background goroutine.
// It optimizes by recalculating the score for a single submission, but recalculates percentiles for all.
func (s *Service) recalculateScoresForSubmissionAsync(examID uint, submissionID uint) {
	ctx := context.Background()

	tx := s.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		fmt.Printf("ERROR: could not begin transaction for async recalculateScoresForSubmission for exam %d submission %d: %v\n", examID, submissionID, tx.Error)
		return
	}
	defer tx.Rollback()

	if err := s.examRepo.RecalculateScoresForSubmission(ctx, tx, examID, submissionID); err != nil {
		fmt.Printf("ERROR: failed to asynchronously recalculate score for exam %d submission %d: %v\n", examID, submissionID, err)
		return
	}

	if err := tx.Commit().Error; err != nil {
		fmt.Printf("ERROR: could not commit transaction for async recalculateScoresForSubmission for exam %d: %v\n", examID, err)
	}
}

func (s *Service) updateUserFromSubmission(user models.ExamUser, req SubmissionUpdateRequest) error {
	updated := false
	if req.Name != "" && user.Name != req.Name {
		user.Name = req.Name
		updated = true
	}
	if req.Surname != "" && user.Surname != req.Surname {
		user.Surname = req.Surname
		updated = true
	}
	if req.Email != "" && user.Email != req.Email {
		user.Email = req.Email
		updated = true
	}
	if req.DNI != "" && user.DNI != req.DNI {
		user.DNI = req.DNI
		updated = true
	}
	if updated && user.ID != 0 {
		if err := s.db.Model(&user).Updates(user).Error; err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) updateAnswersFromSubmission(submission *models.UserExamSubmission, req SubmissionUpdateRequest) error {
	// Update AnswersData (primary)
	answersMap := make(models.AnswersJSON)
	if submission.AnswersData != nil {
		maps.Copy(answersMap, *submission.AnswersData)
	}
	for _, answer := range req.Answers {
		answersMap[answer.QuestionID] = answer.Answer
	}
	submission.AnswersData = &answersMap

	// Phase 1: dual-write to user_answer
	ctx := context.Background()
	var existingAnswers []models.UserAnswer
	if err := s.db.Where("submission_id = ?", submission.ID).Find(&existingAnswers).Error; err != nil {
		return err
	}
	existingMap := make(map[uint]*models.UserAnswer)
	for i := range existingAnswers {
		existingMap[existingAnswers[i].QuestionID] = &existingAnswers[i]
	}
	for _, answer := range req.Answers {
		if existing, ok := existingMap[answer.QuestionID]; ok {
			existing.Answer = answer.Answer
			if err := s.submissionRepo.SaveAnswer(ctx, s.db, existing); err != nil {
				return err
			}
		} else {
			newAnswer := models.UserAnswer{
				SubmissionID: submission.ID,
				QuestionID:   answer.QuestionID,
				Answer:       answer.Answer,
			}
			if err := s.submissionRepo.CreateAnswer(ctx, s.db, &newAnswer); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *Service) ListSubmissions(examID uint, limit, offset int, includeStats bool, search, orderBy, orderDir string, moodleSynced *bool, resultType *string) (*ListSubmissionsResult, error) {
	orderClause := buildSubmissionOrder(strings.TrimSpace(orderBy), strings.TrimSpace(orderDir))

	subs, err := s.submissionRepo.List(context.Background(), s.db, examID, limit, offset, search, orderClause, moodleSynced, resultType)
	if err != nil {
		return nil, err
	}

	result := &ListSubmissionsResult{
		Submissions: subs,
	}
	if includeStats {
		totalCount, err := s.submissionRepo.Count(context.Background(), s.db, examID, moodleSynced, resultType)
		if err != nil {
			return nil, err
		}

		avg, err := s.submissionRepo.GetAverageScore(context.Background(), s.db, examID, moodleSynced, resultType)
		if err != nil {
			return nil, err
		}

		result.TotalSubmissions = totalCount
		result.AverageScore = avg
		result.StatsIncluded = true
	}
	return result, nil
}

type submissionEmailRow struct {
	SubmissionEmail sql.NullString `gorm:"column:submission_email"`
}

func (s *Service) ListSubmissionEmails(examID uint, search, orderBy, orderDir string, moodleSynced *bool) ([]string, error) {
	var rows []submissionEmailRow
	query := s.db.Model(&models.UserExamSubmission{}).
		Select("exam_user.email AS submission_email").
		Joins("LEFT JOIN exam_user ON exam_user.id = user_exam_submission.user_id").
		Where("exam_id = ?", examID)
	if moodleSynced != nil {
		if *moodleSynced {
			query = query.Where("exam_user.moodle_id IS NOT NULL")
		} else {
			query = query.Where("exam_user.moodle_id IS NULL")
		}
	}
	if sanitized := strings.TrimSpace(search); sanitized != "" {
		like := fmt.Sprintf("%%%s%%", strings.ToLower(sanitized))
		query = query.Where(
			"LOWER(COALESCE(exam_user.name, '')) LIKE ? OR "+
				"LOWER(COALESCE(exam_user.surname, '')) LIKE ? OR "+
				"LOWER(COALESCE(exam_user.email, '')) LIKE ? OR "+
				"LOWER(COALESCE(exam_user.dni, '')) LIKE ?",
			like, like, like, like,
		)
	}
	orderClause := buildSubmissionOrder(strings.TrimSpace(orderBy), strings.TrimSpace(orderDir))
	if orderClause != "" {
		query = query.Order(orderClause)
	}
	if err := query.Find(&rows).Error; err != nil {
		return nil, err
	}

	seen := map[string]struct{}{}
	emails := make([]string, 0, len(rows))
	for _, row := range rows {
		for _, candidate := range []sql.NullString{row.SubmissionEmail} {
			email := strings.TrimSpace(candidate.String)
			if email == "" {
				continue
			}
			normalized := strings.ToLower(email)
			if _, ok := seen[normalized]; ok {
				continue
			}
			seen[normalized] = struct{}{}
			emails = append(emails, email)
		}
	}
	return emails, nil
}

func buildSubmissionOrder(orderBy, orderDir string) string {
	order := strings.ToLower(orderBy)
	dir := strings.ToUpper(orderDir)
	if dir != "ASC" && dir != "DESC" {
		dir = ""
	}

	field := "user_exam_submission.submitted_at"
	switch order {
	case "score":
		field = "user_exam_submission.score"
		if dir == "" {
			dir = "DESC"
		}
	case "name":
		field = "exam_user.name"
		if dir == "" {
			dir = "ASC"
		}
	case "surname":
		field = "exam_user.surname"
		if dir == "" {
			dir = "ASC"
		}
	case "submitted_at", "time":
		field = "submitted_at"
		if dir == "" {
			dir = "DESC"
		}
	default:
		if dir == "" {
			dir = "DESC"
		}
	}
	if dir == "" {
		dir = "DESC"
	}
	return fmt.Sprintf("%s %s", field, dir)
}
