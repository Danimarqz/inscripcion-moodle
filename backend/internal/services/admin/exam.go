package admin

import (
	"context"
	"errors"
	"fmt"
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
	return exam, nil
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

	questions, err := createQuestions(req.Questions)
	if err != nil {
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
		SecondaryMaxScores:   req.SecondaryMaxScores,
		PassingCriteriaType:  req.PassingCriteriaType,
		PassingCriteriaValue: req.PassingCriteriaValue,
		ExamWeight:           req.ExamWeight,
		MaxMerits:            req.MaxMerits,
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

	count := 1

	for _, input := range inputs {
		isActive := true
		isCancelled := false
		if input.IsActive != nil {
			isActive = *input.IsActive
		}
		if input.IsCancelled != nil {
			isCancelled = *input.IsCancelled
		}

		name := count
		count++

		normalizedOption := strings.ToUpper(strings.TrimSpace(input.CorrectOption))
		if normalizedOption == "" {
			return nil, ErrInvalidOption
		}

		model := models.Question{
			ID:            0,
			Name:          name,
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

	if req.PassingCriteriaType != nil || req.PassingCriteriaValue != nil {
		exam.PassingThreshold = examservice.ComputePassingThreshold(s.db, exam, s.examRepo)
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

	if req.SubtractsPoints != nil || req.PenaltyValue != nil || req.MaxScore != nil || len(req.Questions) > 0 {
		if err := s.examRepo.RecalculateScores(context.Background(), s.db, exam.ID); err != nil {
			return nil, fmt.Errorf("failed to recalculate scores: %w", err)
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
	count := 1

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

		model.Name = count
		count++

		questions = append(questions, model)
	}

	return questions, nil
}
