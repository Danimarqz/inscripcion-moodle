package admin

import (
	"context"
	"errors"
	"fmt"
	"math"
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
	exam, err := s.examRepo.FindExamByIDLite(context.Background(), s.db, examID)
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
	if s.db != nil {
		s.db.Where("exam_id = ?", examID).Order("position asc").Find(&exam.Groups)
	}
	return exam, nil
}

// GetExamQuestions returns an exam's questions, fetched on demand so the exam
// config read (GetExam) can stay lightweight.
func (s *Service) GetExamQuestions(examID uint) ([]models.Question, error) {
	return s.examRepo.FindQuestionsByExamID(context.Background(), s.db, examID)
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
func validateScoring(mode string, ppc, ppw, wrongBlockSize *float64) error {
	// Block penalty (cada N falladas resta 1 pregunta) must be a whole number >= 1
	// when set. 0 disables it. Applies to legacy flat and grouped scoring; the
	// absolute branch simply ignores it, so no mode restriction is needed.
	if wrongBlockSize != nil && *wrongBlockSize != 0 {
		if *wrongBlockSize < 1 || *wrongBlockSize != math.Trunc(*wrongBlockSize) {
			return ErrInvalidWrongBlockSize
		}
	}
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

	if err := validateScoring(req.ScoringMode, req.PointsPerCorrect, req.PointsPerWrong, req.WrongBlockSize); err != nil {
		return nil, err
	}

	if err := validateGroups(req.Groups, req.Questions); err != nil {
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
		WrongBlockSize:       req.WrongBlockSize,
		SecondaryMaxScores:   req.SecondaryMaxScores,
		PassingCriteriaType:  req.PassingCriteriaType,
		PassingCriteriaValue: req.PassingCriteriaValue,
		ExamWeight:           req.ExamWeight,
		MaxMerits:            req.MaxMerits,
		DisplayExamWeight:    req.DisplayExamWeight,
		SkipWeights:          req.SkipWeights,
		RaffleEnabled:        req.RaffleEnabled,
		RaffleTerms:          req.RaffleTerms,
		Questions:            questions,
		Groups:               buildGroups(req.Groups, 0),
	}

	if len(activeQuestions(questions)) == 0 {
		return nil, ErrActiveQuestions
	}

	if err := s.examRepo.CreateExam(context.Background(), s.db, exam); err != nil {
		return nil, err
	}

	// Groups now have IDs (assigned by the association insert). Link each question
	// to its group by position and persist the group_id.
	if len(exam.Groups) > 0 {
		assignGroupIDs(exam.Questions, req.Questions, exam.Groups)
		for i := range exam.Questions {
			if err := s.db.Model(&models.Question{}).Where("id = ?", exam.Questions[i].ID).
				Update("group_id", exam.Questions[i].GroupID).Error; err != nil {
				return nil, err
			}
		}
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
	if req.WrongBlockSize != nil {
		exam.WrongBlockSize = req.WrongBlockSize
	}
	// Validate the EFFECTIVE post-merge state so switching to absolute mode
	// without (re)sending the points is rejected.
	if err := validateScoring(exam.ScoringMode, exam.PointsPerCorrect, exam.PointsPerWrong, exam.WrongBlockSize); err != nil {
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
	if req.RaffleEnabled != nil {
		exam.RaffleEnabled = *req.RaffleEnabled
	}
	if req.RaffleTerms != nil {
		exam.RaffleTerms = *req.RaffleTerms
	}

	if req.PassingCriteriaType != nil || req.PassingCriteriaValue != nil {
		exam.PassingThreshold = examservice.ComputePassingThreshold(s.db, exam, s.examRepo)
	}

	if len(req.Questions) > 0 {
		if err := validateQuestionNumbers(req.Questions); err != nil {
			return nil, err
		}
	}

	if err := validateGroups(req.Groups, req.Questions); err != nil {
		return nil, err
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

	// Reconcile question groups BEFORE saving questions: delete removed groups,
	// upsert the rest so new groups get IDs, then link each question to its group
	// by position. Going from grouped to flat (empty req.Groups while the exam had
	// groups) deletes all groups and clears every question's group_id. Skipped
	// entirely for flat-stays-flat exams so the path never touches the DB.
	if len(req.Groups) > 0 || len(exam.Groups) > 0 {
		desiredGroups := buildGroups(req.Groups, exam.ID)
		keepIDs := make([]uint, 0, len(desiredGroups))
		for _, g := range desiredGroups {
			if g.ID != 0 {
				keepIDs = append(keepIDs, g.ID)
			}
		}
		delQ := s.db.Where("exam_id = ?", exam.ID)
		if len(keepIDs) > 0 {
			delQ = delQ.Where("id NOT IN ?", keepIDs)
		}
		if err := delQ.Delete(&models.QuestionGroup{}).Error; err != nil {
			return nil, err
		}
		for i := range desiredGroups {
			if err := s.db.Save(&desiredGroups[i]).Error; err != nil {
				return nil, err
			}
		}
		assignGroupIDs(updatedQuestions, req.Questions, desiredGroups)
		exam.Groups = desiredGroups
	}

	exam.Questions = updatedQuestions
	if err := s.examRepo.UpdateExam(context.Background(), s.db, exam); err != nil {
		return nil, err
	}

	if req.SubtractsPoints != nil || req.PenaltyValue != nil || req.MaxScore != nil ||
		req.ScoringMode != nil || req.PointsPerCorrect != nil || req.PointsPerWrong != nil ||
		req.WrongBlockSize != nil || len(req.Questions) > 0 || len(req.Groups) > 0 {
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

// validateGroups checks group config and that, when groups exist, every active
// non-cancelled question is assigned to an existing group position. Question
// numbering stays GLOBAL 1..N (validateQuestionNumbers) — groups only partition,
// they do not renumber, so the (exam_id, name) unique constraint still holds.
func validateGroups(groups []QuestionGroupInput, questions []QuestionInput) error {
	if len(groups) == 0 {
		for _, q := range questions {
			if q.GroupPosition != nil {
				return ErrInvalidGroups
			}
		}
		return nil
	}
	positions := make(map[int]bool, len(groups))
	for _, g := range groups {
		if strings.TrimSpace(g.Name) == "" || g.MaxScore <= 0 || g.PointsPerWrong < 0 {
			return ErrInvalidGroups
		}
		if g.MinPassingScore != nil && (*g.MinPassingScore < 0 || *g.MinPassingScore > g.MaxScore) {
			return ErrInvalidGroups
		}
		if positions[g.Position] {
			return ErrInvalidGroups
		}
		positions[g.Position] = true
	}
	for _, q := range questions {
		active := q.IsActive == nil || *q.IsActive
		cancelled := q.IsCancelled != nil && *q.IsCancelled
		if !active || cancelled {
			continue
		}
		if q.GroupPosition == nil || !positions[*q.GroupPosition] {
			return ErrInvalidGroups
		}
	}
	return nil
}

// buildGroups maps group inputs to models, carrying the ID for updates.
func buildGroups(inputs []QuestionGroupInput, examID uint) []models.QuestionGroup {
	groups := make([]models.QuestionGroup, 0, len(inputs))
	for _, in := range inputs {
		g := models.QuestionGroup{
			ExamID:          examID,
			Name:            strings.TrimSpace(in.Name),
			Position:        in.Position,
			MaxScore:        in.MaxScore,
			PointsPerWrong:  in.PointsPerWrong,
			MinPassingScore: in.MinPassingScore,
			Eliminatory:     in.Eliminatory,
		}
		if in.ID != nil {
			g.ID = *in.ID
		}
		groups = append(groups, g)
	}
	return groups
}

// assignGroupIDs sets each question's GroupID from its input GroupPosition using
// the saved groups' position->id map. questions and inputs share the same order.
func assignGroupIDs(questions []models.Question, inputs []QuestionInput, groups []models.QuestionGroup) {
	posToID := make(map[int]uint, len(groups))
	for _, g := range groups {
		posToID[g.Position] = g.ID
	}
	for i := range questions {
		questions[i].GroupID = nil
		if i < len(inputs) && inputs[i].GroupPosition != nil {
			if id, ok := posToID[*inputs[i].GroupPosition]; ok {
				gid := id
				questions[i].GroupID = &gid
			}
		}
	}
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
