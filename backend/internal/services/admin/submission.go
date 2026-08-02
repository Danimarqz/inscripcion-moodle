package admin

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"maps"
	"strings"
	"time"

	"github.com/inscripcion-moodle/go-backend/internal/models"
	"github.com/inscripcion-moodle/go-backend/internal/scoring"
	examservice "github.com/inscripcion-moodle/go-backend/internal/services/exam"
)

// recalcSem limits concurrent recalculate goroutines. Max 5.
var recalcSem = make(chan struct{}, 5)

func (s *Service) DeleteSubmission(submissionID uint) (examID uint, err error) {
	queryCtx, queryCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer queryCancel()
	submission, err := s.submissionRepo.FindByID(queryCtx, s.db, submissionID)
	if err != nil {
		return 0, err
	}
	queryCtx, queryCancel = context.WithTimeout(context.Background(), 30*time.Second)
	defer queryCancel()
	if err := s.submissionRepo.Delete(queryCtx, s.db, submissionID); err != nil {
		return 0, err
	}
	return submission.ExamID, nil
}

func (s *Service) GetSubmission(submissionID uint) (*models.UserExamSubmission, error) {
	queryCtx, queryCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer queryCancel()
	return s.submissionRepo.FindByID(queryCtx, s.db, submissionID)
}

// GetSubmissionBreakdown computes the score breakdown for a single submission,
// reusing the same scoring functions and grouped/Xunta mode selection as the
// student-facing buildSubmissionPayload (via fetchScoreBreakdownFromDB). Answers
// are read exclusively from submission.AnswersData (jsonb). Returns nil when the
// exam has no active, non-cancelled questions.
func (s *Service) GetSubmissionBreakdown(submissionID uint) (*SubmissionBreakdown, error) {
	queryCtx, queryCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer queryCancel()
	submission, err := s.submissionRepo.FindByID(queryCtx, s.db, submissionID)
	if err != nil {
		return nil, err
	}

	queryCtx, queryCancel = context.WithTimeout(context.Background(), 30*time.Second)
	defer queryCancel()
	exam, err := s.examRepo.FindExamByID(queryCtx, s.db, submission.ExamID)
	if err != nil {
		return nil, err
	}

	return computeSubmissionBreakdown(exam, submission)
}

// computeSubmissionBreakdown is the pure (no-DB) breakdown computation shared by
// the per-submission detail endpoint and the list endpoint. The exam must be
// loaded with Questions + Groups; answers are read from submission.AnswersData.
func computeSubmissionBreakdown(exam *models.Exam, submission *models.UserExamSubmission) (*SubmissionBreakdown, error) {
	var err error
	// Match fetchScoreBreakdownFromDB: only active, non-cancelled questions count.
	questions := make([]models.Question, 0, len(exam.Questions))
	for _, q := range exam.Questions {
		if q.IsActive && !q.IsCancelled {
			questions = append(questions, q)
		}
	}
	if len(questions) == 0 {
		return nil, nil
	}

	answerMap := make(map[uint]string)
	if submission.AnswersData != nil {
		answerMap = map[uint]string(*submission.AnswersData)
	}

	// Same grouped/Xunta selection as fetchScoreBreakdownFromDB.
	var bd *examservice.ScoreBreakdown
	groups := exam.Groups
	if len(groups) > 0 {
		if exam.ScoringMode == "xunta" {
			bd, err = examservice.CalculateGroupedXuntaBreakdown(groups, questions, answerMap)
		} else {
			bd, err = examservice.CalculateGroupedBreakdown(groups, questions, answerMap, examservice.ScoringConfigFromExam(exam).WrongBlockSize)
		}
	} else {
		bd, err = examservice.CalculateScoreBreakdownCfg(questions, answerMap, examservice.ScoringConfigFromExam(exam))
	}
	if err != nil {
		return nil, err
	}
	if bd == nil {
		return nil, nil
	}

	// Global pass: grouped exams pass iff every eliminatory group met its
	// minimum; flat exams use the exam's passing threshold vs the stored score
	// (same rule as buildSubmissionPayload's evaluatePassStatus).
	var isPassed *bool
	if len(bd.Groups) > 0 {
		p := scoring.GroupedResult{Groups: bd.Groups}.AllEliminatoryPassed()
		isPassed = &p
	} else {
		isPassed = evaluateFlatPassStatus(submission.Score, exam.PassingThreshold)
	}

	return &SubmissionBreakdown{
		Score:            submission.Score,
		CorrectAnswers:   bd.CorrectAnswers,
		IncorrectAnswers: bd.IncorrectAnswers,
		NotAnswered:      bd.NotAnswered,
		TotalQuestions:   bd.TotalQuestions,
		IsPassed:         isPassed,
		Groups:           bd.Groups,
	}, nil
}

// evaluateFlatPassStatus mirrors exam.evaluatePassStatus: nil threshold → nil
// (no pass criteria), nil score → false, otherwise score >= threshold. The
// scoring math lives in internal/scoring; this is only the pass/fail gate.
func evaluateFlatPassStatus(score *float64, threshold *float64) *bool {
	if threshold == nil {
		return nil
	}
	if score == nil {
		p := false
		return &p
	}
	p := *score >= *threshold
	return &p
}

func (s *Service) UpdateSubmission(submissionID uint, req SubmissionUpdateRequest) (*models.UserExamSubmission, error) {
	queryCtx, queryCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer queryCancel()
	submission, err := s.submissionRepo.FindByID(queryCtx, s.db, submissionID)
	if err != nil {
		return nil, err
	}

	if err := s.updateUserFromSubmission(&submission.User, req); err != nil {
		return nil, err
	}

	if err := s.updateAnswersFromSubmission(submission, req); err != nil {
		return nil, err
	}

	if req.Merits != nil {
		submission.Merits = req.Merits
	}

	queryCtx, queryCancel = context.WithTimeout(context.Background(), 30*time.Second)
	defer queryCancel()
	if err := s.submissionRepo.Update(queryCtx, s.db, submission); err != nil {
		return nil, err
	}

	go s.recalculateScoresForSubmissionAsync(submission.ExamID, submission.ID)

	queryCtx, queryCancel = context.WithTimeout(context.Background(), 30*time.Second)
	defer queryCancel()
	return s.submissionRepo.FindByID(queryCtx, s.db, submissionID)
}

// recalculateScoresForSubmissionAsync runs the score and percentile recalculation in a separate
// transaction and context, making it safe to run in a background goroutine.
// It optimizes by recalculating the score for a single submission, but recalculates percentiles for all.
func (s *Service) recalculateScoresForSubmissionAsync(examID uint, submissionID uint) {
	recalcSem <- struct{}{}
	defer func() { <-recalcSem }()

	ctx := context.Background()

	tx := s.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		log.Printf("ERROR: could not begin transaction for async recalculateScoresForSubmission for exam %d submission %d: %v", examID, submissionID, tx.Error)
		return
	}
	defer tx.Rollback()

	if err := s.examRepo.RecalculateScoresForSubmission(ctx, tx, examID, submissionID); err != nil {
		log.Printf("ERROR: failed to asynchronously recalculate score for exam %d submission %d: %v", examID, submissionID, err)
		return
	}

	if err := tx.Commit().Error; err != nil {
		log.Printf("ERROR: could not commit transaction for async recalculateScoresForSubmission for exam %d: %v", examID, err)
	}
}

func (s *Service) updateUserFromSubmission(user *models.ExamUser, req SubmissionUpdateRequest) error {
	updates := make(map[string]any)
	if req.Name != "" && user.Name != req.Name {
		user.Name = req.Name
		updates["name"] = req.Name
	}
	if req.Surname != "" && user.Surname != req.Surname {
		user.Surname = req.Surname
		updates["surname"] = req.Surname
	}
	if req.Email != "" && user.Email != req.Email {
		user.Email = req.Email
		updates["email"] = req.Email
	}
	if req.DNI != "" && user.DNI != req.DNI {
		user.DNI = req.DNI
		updates["dni"] = req.DNI
	}
	if len(updates) > 0 && user.ID != 0 {
		if err := s.db.Model(&models.ExamUser{ID: user.ID}).Updates(updates).Error; err != nil {
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
	// AnswersData is persisted by the caller (submissionRepo.Update). The legacy
	// user_answer dual-write was removed — nothing reads it as a source of truth.
	return nil
}

func (s *Service) ListSubmissions(examID uint, limit, offset int, includeStats bool, search, orderBy, orderDir string, moodleSynced *bool, resultType *string) (*ListSubmissionsResult, error) {
	orderClause := buildSubmissionOrder(strings.TrimSpace(orderBy), strings.TrimSpace(orderDir))

	// includeAnswers=true so we can compute the per-row breakdown server-side.
	// Answers are stripped from each item before serialization (kept off the wire).
	queryCtx, queryCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer queryCancel()
	subs, err := s.submissionRepo.List(queryCtx, s.db, examID, limit, offset, search, orderClause, moodleSynced, resultType, true)
	if err != nil {
		return nil, err
	}

	items := make([]AdminSubmissionListItem, 0, len(subs))
	if len(subs) > 0 {
		queryCtx, queryCancel = context.WithTimeout(context.Background(), 30*time.Second)
		defer queryCancel()
		exam, err := s.examRepo.FindExamByID(queryCtx, s.db, examID)
		if err != nil {
			return nil, err
		}
		for i := range subs {
			sub := subs[i]
			breakdown, bdErr := computeSubmissionBreakdown(exam, &sub)
			if bdErr != nil {
				log.Printf("failed to compute breakdown for submission %d: %v", sub.ID, bdErr)
			}
			sub.AnswersData = nil // strip heavy answers from the list payload
			items = append(items, AdminSubmissionListItem{
				UserExamSubmission: sub,
				Breakdown:          breakdown,
			})
		}
	}

	result := &ListSubmissionsResult{
		Submissions: items,
	}
	if includeStats {
		queryCtx, queryCancel = context.WithTimeout(context.Background(), 30*time.Second)
		defer queryCancel()
		totalCount, err := s.submissionRepo.Count(queryCtx, s.db, []uint{examID}, moodleSynced, resultType)
		if err != nil {
			return nil, err
		}

		queryCtx, queryCancel = context.WithTimeout(context.Background(), 30*time.Second)
		defer queryCancel()
		avg, avgOfficial, _, err := s.submissionRepo.GetAverageScore(queryCtx, s.db, examID, moodleSynced, resultType, false)
		if err != nil {
			return nil, err
		}

		result.TotalSubmissions = totalCount
		result.AverageScore = avg
		result.AverageScoreOfficial = avgOfficial
		result.StatsIncluded = true

		// Always compute the group aggregate too (cheap) so the "compare against
		// group" toggle is a client-side switch — no extra round-trip per toggle.
		queryCtx, queryCancel = context.WithTimeout(context.Background(), 30*time.Second)
		defer queryCancel()
		groupAvg, groupOfficial, examIDs, err := s.submissionRepo.GetAverageScore(queryCtx, s.db, examID, moodleSynced, resultType, true)
		if err != nil {
			return nil, err
		}
		if len(examIDs) > 1 {
			queryCtx, queryCancel = context.WithTimeout(context.Background(), 30*time.Second)
			defer queryCancel()
			groupCount, err := s.submissionRepo.Count(queryCtx, s.db, examIDs, moodleSynced, resultType)
			if err != nil {
				return nil, err
			}
			result.GroupAverageScore = groupAvg
			result.GroupAverageScoreOfficial = groupOfficial
			result.GroupTotalSubmissions = &groupCount
			var names []string
			if err := s.db.Model(&models.Exam{}).Where("id IN ?", examIDs).
				Order("name ASC").Pluck("name", &names).Error; err != nil {
				return nil, err
			}
			result.GroupExamNames = names
		}
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
