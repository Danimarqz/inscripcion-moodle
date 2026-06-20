package admin

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"gorm.io/gorm"

	"github.com/inscripcion-moodle/go-backend/internal/models"
	examservice "github.com/inscripcion-moodle/go-backend/internal/services/exam"
)

func (s *Service) ListExams() ([]models.Exam, error) {
	return s.examRepo.ListExams(context.Background(), s.db)
}

func (s *Service) GetExam(examID uint) (*models.Exam, error) {
	exam, err := s.examRepo.FindExamByID(context.Background(), s.db, examID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrExamNotFound
		}
		return nil, err
	}
	if exam.PercentileGroup != nil {
		s.db.Model(&models.Exam{}).
			Where("percentile_group = ? AND id <> ?", *exam.PercentileGroup, examID).
			Pluck("id", &exam.AssociatedExamIDs)
	}
	return exam, nil
}

// normalizeMode maps an empty scoring mode to the legacy default so the
// in-memory value is explicit (GORM's column default only applies at the DB).
func normalizeMode(s string) string {
	if s == "" {
		return "legacy"
	}
	return s
}

// validateScoring rejects unknown modes and incomplete/negative absolute config.
// points_per_wrong == 0 is valid (no deduction); ppw > ppc is allowed by design
// (a wrong may cost more than a right earns).
func validateScoring(mode string, ppc, ppw *float64) error {
	switch mode {
	case "", "legacy":
		return nil
	case "absolute":
		if ppc == nil || ppw == nil || *ppc < 0 || *ppw < 0 {
			return ErrAbsoluteScoringConfig
		}
		return nil
	default:
		return ErrInvalidScoringMode
	}
}

func (s *Service) CreateExam(req CreateExamRequest) (*models.Exam, error) {
	if len(req.Questions) == 0 {
		return nil, ErrExamNoQuestions
	}

	exists, err := s.examRepo.CountByName(context.Background(), s.db, req.Name)
	if err != nil {
		return nil, err
	}
	if exists > 0 {
		return nil, ErrExamNameConflict
	}

	if err := validateQuestionNumbers(req.Questions); err != nil {
		return nil, err
	}

	questions, err := createQuestions(req.Questions)
	if err != nil {
		return nil, err
	}

	if req.DisplayExamWeight != nil && (*req.DisplayExamWeight < 0 || *req.DisplayExamWeight > 1) {
		return nil, ErrInvalidDisplayWeight
	}

	if err := validateScoring(req.ScoringMode, req.PointsPerCorrect, req.PointsPerWrong); err != nil {
		return nil, err
	}

	exam := &models.Exam{
		Name:                 req.Name,
		IsActive:             req.IsActive,
		ShowScore:            req.ShowScore,
		ShowPercentile:       req.ShowPercentile,
		ShowScoreFull:        req.ShowScoreFull,
		ValidatedTribunal:    req.ValidatedTribunal,
		SubtractsPoints:      req.SubtractsPoints,
		PenaltyValue:         req.PenaltyValue,
		MaxScore:             req.MaxScore,
		ScoringMode:          normalizeMode(req.ScoringMode),
		PointsPerCorrect:     req.PointsPerCorrect,
		PointsPerWrong:       req.PointsPerWrong,
		SecondaryMaxScores:   req.SecondaryMaxScores,
		PassingCriteriaType:  req.PassingCriteriaType,
		PassingCriteriaValue: req.PassingCriteriaValue,
		ExamWeight:           req.ExamWeight,
		MaxMerits:            req.MaxMerits,
		DisplayExamWeight:    req.DisplayExamWeight,
		SkipWeights:          req.SkipWeights,
		UseOfficialScores:    req.UseOfficialScores,
		Questions:            questions,
	}

	if len(activeQuestions(questions)) == 0 {
		return nil, ErrActiveQuestions
	}

	if err := s.examRepo.CreateExam(context.Background(), s.db, exam); err != nil {
		return nil, err
	}
	return exam, nil
}

func createQuestions(inputs []QuestionInput) ([]models.Question, error) {
	questions := make([]models.Question, 0, len(inputs))

	for _, input := range inputs {
		isActive := true
		isCancelled := false
		if input.IsActive != nil {
			isActive = *input.IsActive
		}
		if input.IsCancelled != nil {
			isCancelled = *input.IsCancelled
		}

		normalizedOption := strings.ToUpper(strings.TrimSpace(input.CorrectOption))
		if normalizedOption == "" {
			return nil, ErrInvalidOption
		}

		model := models.Question{
			ID:            0,
			Name:          *input.Name,
			Label:         normalizeLabel(input.Label),
			IsActive:      isActive,
			IsCancelled:   isCancelled,
			CorrectOption: normalizedOption,
		}
		if input.ID != nil {
			model.ID = *input.ID
		}
		questions = append(questions, model)
	}
	return questions, nil
}

// validateQuestionNumbers ensures every QuestionInput has an explicit positive
// Name and that the set of names forms a contiguous 1..N sequence (no gaps, no
// duplicates). This is the single source of truth for the contiguity rule; the
// (exam_id, name) unique constraint at the DB level is a belt-and-braces backup.
func validateQuestionNumbers(inputs []QuestionInput) error {
	if len(inputs) == 0 {
		return nil
	}
	names := make([]int, 0, len(inputs))
	for _, input := range inputs {
		if input.Name == nil || *input.Name <= 0 {
			return ErrInvalidQuestionNumbers
		}
		names = append(names, *input.Name)
	}
	sort.Ints(names)
	for i, n := range names {
		if n != i+1 {
			return ErrInvalidQuestionNumbers
		}
	}
	return nil
}

func (s *Service) UpdateExam(examID uint, req EditExamRequest) (*models.Exam, error) {
	exam, err := s.examRepo.FindExamByID(context.Background(), s.db, examID)
	if err != nil {
		return nil, err
	}

	if req.Name != nil && *req.Name != "" {
		exam.Name = *req.Name
	}
	if req.IsActive != nil {
		exam.IsActive = *req.IsActive
	}
	if req.ShowScore != nil {
		exam.ShowScore = *req.ShowScore
	}
	if req.ShowPercentile != nil {
		exam.ShowPercentile = *req.ShowPercentile
	}
	if req.ShowScoreFull != nil {
		exam.ShowScoreFull = *req.ShowScoreFull
	}
	if req.ValidatedTribunal != nil {
		exam.ValidatedTribunal = *req.ValidatedTribunal
	}
	if req.SubtractsPoints != nil {
		exam.SubtractsPoints = *req.SubtractsPoints
	}
	if req.PenaltyValue != nil {
		exam.PenaltyValue = req.PenaltyValue
	}
	if req.MaxScore != nil {
		exam.MaxScore = req.MaxScore
	}
	if req.ScoringMode != nil {
		exam.ScoringMode = normalizeMode(*req.ScoringMode)
	}
	if req.PointsPerCorrect != nil {
		exam.PointsPerCorrect = req.PointsPerCorrect
	}
	if req.PointsPerWrong != nil {
		exam.PointsPerWrong = req.PointsPerWrong
	}
	// Validate the EFFECTIVE post-merge state so switching to absolute mode
	// without (re)sending the points is rejected.
	if err := validateScoring(exam.ScoringMode, exam.PointsPerCorrect, exam.PointsPerWrong); err != nil {
		return nil, err
	}
	if req.SecondaryMaxScores != nil {
		exam.SecondaryMaxScores = *req.SecondaryMaxScores
	}
	if req.PassingCriteriaType != nil {
		exam.PassingCriteriaType = *req.PassingCriteriaType
	}
	if req.PassingCriteriaValue != nil {
		exam.PassingCriteriaValue = req.PassingCriteriaValue
	}

	if req.ExamWeight != nil {
		if *req.ExamWeight < 0 || *req.ExamWeight > 1 {
			return nil, ErrInvalidExamWeight
		}
		exam.ExamWeight = *req.ExamWeight
	}
	if req.MaxMerits != nil {
		if *req.MaxMerits <= 0 {
			return nil, ErrInvalidMaxMerits
		}
		exam.MaxMerits = *req.MaxMerits
	}
	if req.ClearDisplayWeight != nil && *req.ClearDisplayWeight {
		exam.DisplayExamWeight = nil
	} else if req.DisplayExamWeight != nil {
		if *req.DisplayExamWeight < 0 || *req.DisplayExamWeight > 1 {
			return nil, ErrInvalidDisplayWeight
		}
		exam.DisplayExamWeight = req.DisplayExamWeight
	}
	if req.SkipWeights != nil {
		exam.SkipWeights = *req.SkipWeights
	}
	if req.UseOfficialScores != nil {
		exam.UseOfficialScores = *req.UseOfficialScores
	}

	if req.PassingCriteriaType != nil || req.PassingCriteriaValue != nil {
		exam.PassingThreshold = examservice.ComputePassingThreshold(s.db, exam, s.examRepo)
	}

	if len(req.Questions) > 0 {
		if err := validateQuestionNumbers(req.Questions); err != nil {
			return nil, err
		}
	}

	inputQuestionIDs := make(map[uint]struct{})
	for _, q := range req.Questions {
		if q.ID != nil {
			inputQuestionIDs[*q.ID] = struct{}{}
		}
	}

	toDelete := []uint{}
	for _, q := range exam.Questions {
		if _, found := inputQuestionIDs[q.ID]; !found {
			toDelete = append(toDelete, q.ID)
		}
	}

	if len(toDelete) > 0 {
		if err := s.questionRepo.DeleteQuestions(context.Background(), s.db, toDelete); err != nil {
			return nil, err
		}
	}

	updatedQuestions, err := s.mergeQuestions(exam.ID, exam.Questions, req.Questions)
	if err != nil {
		return nil, err
	}

	if len(activeQuestions(updatedQuestions)) == 0 {
		return nil, ErrActiveQuestions
	}

	exam.Questions = updatedQuestions
	if err := s.examRepo.UpdateExam(context.Background(), s.db, exam); err != nil {
		return nil, err
	}

	if req.SubtractsPoints != nil || req.PenaltyValue != nil || req.MaxScore != nil ||
		req.ScoringMode != nil || req.PointsPerCorrect != nil || req.PointsPerWrong != nil ||
		len(req.Questions) > 0 {
		if err := s.examRepo.RecalculateScores(context.Background(), s.db, exam.ID); err != nil {
			return nil, fmt.Errorf("failed to recalculate scores: %w", err)
		}
	}

	if req.AssociatedExamIDs != nil {
		affected, err := s.examRepo.SetPercentileGroup(context.Background(), s.db, exam.ID, *req.AssociatedExamIDs)
		if err != nil {
			return nil, fmt.Errorf("failed to set percentile group: %w", err)
		}
		for _, id := range affected {
			if err := s.examRepo.RecalculatePercentiles(context.Background(), s.db, id); err != nil {
				return nil, fmt.Errorf("failed to recalculate percentiles: %w", err)
			}
		}
	}

	return exam, nil
}

func (s *Service) DeleteExam(examID uint) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		ctx := context.Background()
		if err := s.submissionRepo.DeleteByExamID(ctx, tx, examID); err != nil {
			return err
		}
		if err := s.questionRepo.DeleteByExamID(ctx, tx, examID); err != nil {
			return err
		}
		if err := s.officialRepo.DeleteByExamID(ctx, tx, examID); err != nil {
			return err
		}
		return s.examRepo.DeleteExam(ctx, tx, examID)
	})
}

func activeQuestions(questions []models.Question) []models.Question {
	out := make([]models.Question, 0, len(questions))
	for _, question := range questions {
		if question.IsActive && !question.IsCancelled {
			out = append(out, question)
		}
	}
	return out
}

func (s *Service) mergeQuestions(examID uint, existing []models.Question, inputs []QuestionInput) ([]models.Question, error) {
	existingMap := map[uint]*models.Question{}
	for i := range existing {
		existingMap[existing[i].ID] = &existing[i]
	}

	questions := make([]models.Question, 0, len(inputs))

	for _, input := range inputs {
		var model models.Question
		if input.ID != nil {
			existingQuestion, ok := existingMap[*input.ID]
			if !ok {
				return nil, ErrQuestionNotFound
			}
			model = *existingQuestion
		} else {
			model = models.Question{ExamID: examID}
		}

		model.CorrectOption = strings.ToUpper(strings.TrimSpace(input.CorrectOption))
		model.IsActive = true
		model.IsCancelled = false
		if input.IsActive != nil {
			model.IsActive = *input.IsActive
		}
		if input.IsCancelled != nil {
			model.IsCancelled = *input.IsCancelled
		}
		model.ExamID = examID

		model.Name = *input.Name
		model.Label = normalizeLabel(input.Label)

		questions = append(questions, model)
	}

	return questions, nil
}
