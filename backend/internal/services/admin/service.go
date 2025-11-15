package admin

import (
	"errors"
	"fmt"
	"strings"

	"gorm.io/gorm"

	"github.com/inscripcion-moodle/go-backend/internal/models"
)

var (
	ErrExamNameConflict = errors.New("ya existe un examen con ese nombre")
	ErrActiveQuestions  = errors.New("el examen debe tener al menos una pregunta activa no anulada")
	ErrQuestionNotFound = errors.New("la pregunta no pertenece al examen")
)

type Service struct {
	db *gorm.DB
}

func New(db *gorm.DB) *Service {
	return &Service{db: db}
}

func (s *Service) ListExams() ([]models.Exam, error) {
	var exams []models.Exam
	if err := s.db.Preload("Questions").Find(&exams).Error; err != nil {
		return nil, err
	}
	return exams, nil
}

func (s *Service) GetExam(examID uint) (*models.Exam, error) {
	var exam models.Exam
	if err := s.db.Preload("Questions").First(&exam, examID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("examen %d no existe: %w", examID, err)
		}
		return nil, err
	}
	return &exam, nil
}

func (s *Service) CreateExam(req CreateExamRequest) (*models.Exam, error) {
	if len(req.Questions) == 0 {
		return nil, errors.New("el examen debe tener al menos una pregunta")
	}

	var exists int64
	if err := s.db.Model(&models.Exam{}).Where("name = ?", req.Name).Count(&exists).Error; err != nil {
		return nil, err
	}
	if exists > 0 {
		return nil, ErrExamNameConflict
	}

	questions, err := buildQuestionModels(req.Questions, nil)
	if err != nil {
		return nil, err
	}

	exam := &models.Exam{
		Name:              req.Name,
		IsActive:          req.IsActive,
		ShowScore:         req.ShowScore,
		ShowPercentile:    req.ShowPercentile,
		ShowScoreFull:     req.ShowScoreFull,
		ValidatedTribunal: req.ValidatedTribunal,
		Questions:         questions,
	}

	if len(activeQuestions(questions)) == 0 {
		return nil, ErrActiveQuestions
	}

	if err := s.db.Create(exam).Error; err != nil {
		return nil, err
	}
	return exam, nil
}

func (s *Service) UpdateExam(examID uint, req EditExamRequest) (*models.Exam, error) {
	var exam models.Exam
	if err := s.db.Preload("Questions").First(&exam, examID).Error; err != nil {
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

	nameGen := newQuestionNameGenerator(questionNames(exam.Questions))
	questions, err := s.mergeQuestions(exam.ID, exam.Questions, req.Questions, nameGen)
	if err != nil {
		return nil, err
	}

	if len(activeQuestions(questions)) == 0 {
		return nil, ErrActiveQuestions
	}

	exam.Questions = questions
	if err := s.db.Session(&gorm.Session{FullSaveAssociations: true}).Save(&exam).Error; err != nil {
		return nil, err
	}

	return &exam, nil
}

func (s *Service) DeleteExam(examID uint) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		var submissionIDs []uint
		if err := tx.Model(&models.UserExamSubmission{}).
			Where("exam_id = ?", examID).
			Pluck("id", &submissionIDs).Error; err != nil {
			return err
		}

		if len(submissionIDs) > 0 {
			if err := tx.Where("submission_id IN (?)", submissionIDs).Delete(&models.UserAnswer{}).Error; err != nil {
				return err
			}
			if err := tx.Where("id IN (?)", submissionIDs).Delete(&models.UserExamSubmission{}).Error; err != nil {
				return err
			}
		}

		if err := tx.Where("exam_id = ?", examID).Delete(&models.Question{}).Error; err != nil {
			return err
		}
		if err := tx.Delete(&models.Exam{}, examID).Error; err != nil {
			return err
		}
		return nil
	})
}

func (s *Service) DeleteSubmission(submissionID uint) (examID uint, err error) {
	var submission models.UserExamSubmission
	if err := s.db.First(&submission, submissionID).Error; err != nil {
		return 0, err
	}
	if err := s.db.Where("submission_id = ?", submissionID).Delete(&models.UserAnswer{}).Error; err != nil {
		return 0, err
	}
	if err := s.db.Delete(&submission).Error; err != nil {
		return 0, err
	}
	return submission.ExamID, nil
}

func (s *Service) ListSubmissions(examID uint, limit, offset int) ([]models.UserExamSubmission, error) {
	var subs []models.UserExamSubmission
	query := s.db.Preload("User").Preload("Answers").
		Where("exam_id = ?", examID).
		Order("submitted_at DESC")
	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}
	if err := query.Find(&subs).Error; err != nil {
		return nil, err
	}
	return subs, nil
}

func (s *Service) ListOfficialResults(examID uint) ([]models.ExamOfficialResult, error) {
	var results []models.ExamOfficialResult
	if err := s.db.Preload("User").
		Where("exam_id = ?", examID).
		Order("apellido_1 ASC, apellido_2 ASC, nombre ASC").
		Find(&results).Error; err != nil {
		return nil, err
	}
	return results, nil
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

func buildQuestionModels(inputs []QuestionInput, existing []models.Question) ([]models.Question, error) {
	var base []int
	for _, q := range existing {
		if q.Name > 0 {
			base = append(base, q.Name)
		}
	}
	nameGen := newQuestionNameGenerator(base)
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
			return nil, errors.New("opcion de respuesta no valida")
		}

		model := models.Question{
			ID:            0,
			Name:          nameGen.Next(input.Name),
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

type questionNameGenerator struct {
	used map[int]struct{}
	next int
}

func newQuestionNameGenerator(existing []int) *questionNameGenerator {
	used := map[int]struct{}{}
	max := 0
	for _, v := range existing {
		if v > 0 {
			used[v] = struct{}{}
			if v > max {
				max = v
			}
		}
	}
	return &questionNameGenerator{
		used: used,
		next: max + 1,
	}
}

func (g *questionNameGenerator) Next(preferred *int) int {
	if preferred != nil && *preferred > 0 {
		if _, ok := g.used[*preferred]; !ok {
			g.used[*preferred] = struct{}{}
			return *preferred
		}
	}

	for {
		candidate := g.next
		g.next++
		if _, ok := g.used[candidate]; !ok {
			g.used[candidate] = struct{}{}
			return candidate
		}
	}
}

func questionNames(questions []models.Question) []int {
	names := make([]int, 0, len(questions))
	for _, q := range questions {
		if q.Name > 0 {
			names = append(names, q.Name)
		}
	}
	return names
}

func (s *Service) mergeQuestions(examID uint, existing []models.Question, inputs []QuestionInput, nameGen *questionNameGenerator) ([]models.Question, error) {
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

		model.Name = nameGen.Next(input.Name)
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

		questions = append(questions, model)
	}

	return questions, nil
}
