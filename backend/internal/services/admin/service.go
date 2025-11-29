package admin

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"gorm.io/gorm"

	"github.com/inscripcion-moodle/go-backend/internal/models"
)

var (
	ErrExamNameConflict      = errors.New("ya existe un examen con ese nombre")
	ErrActiveQuestions       = errors.New("el examen debe tener al menos una pregunta activa no anulada")
	ErrQuestionNotFound      = errors.New("la pregunta no pertenece al examen")
	ErrOfficialResultExists  = errors.New("ya existe un resultado oficial con ese DNI para este examen")
	ErrInvalidOfficialResult = errors.New("datos de resultado oficial invalidos")
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

func (s *Service) UpdateSubmission(submissionID uint, req SubmissionUpdateRequest) (*models.UserExamSubmission, error) {
	var submission models.UserExamSubmission
	if err := s.db.Preload("User").Preload("Answers").First(&submission, submissionID).Error; err != nil {
		return nil, err
	}

	updated := false
	if req.Name != "" && submission.User.Name != req.Name {
		submission.User.Name = req.Name
		updated = true
	}
	if req.Surname != "" && submission.User.Surname != req.Surname {
		submission.User.Surname = req.Surname
		updated = true
	}
	if req.Email != "" && submission.User.Email != req.Email {
		submission.User.Email = req.Email
		updated = true
	}
	if req.DNI != "" && submission.User.DNI != req.DNI {
		submission.User.DNI = req.DNI
		updated = true
	}
	if updated && submission.User.ID != 0 {
		if err := s.db.Model(&submission.User).Updates(submission.User).Error; err != nil {
			return nil, err
		}
	}

	if err := s.db.Save(&submission).Error; err != nil {
		return nil, err
	}

	answerMap := make(map[uint]*models.UserAnswer)
	for i := range submission.Answers {
		answer := submission.Answers[i]
		answerMap[answer.QuestionID] = &submission.Answers[i]
	}

	for _, answer := range req.Answers {
		if existing, ok := answerMap[answer.QuestionID]; ok {
			existing.Answer = answer.Answer
			if err := s.db.Save(existing).Error; err != nil {
				return nil, err
			}
		} else {
			newAnswer := models.UserAnswer{
				SubmissionID: submission.ID,
				QuestionID:   answer.QuestionID,
				Answer:       answer.Answer,
			}
			if err := s.db.Create(&newAnswer).Error; err != nil {
				return nil, err
			}
		}
	}

	if err := s.db.Preload("User").Preload("Answers").First(&submission, submission.ID).Error; err != nil {
		return nil, err
	}
	return &submission, nil
}

func (s *Service) ListSubmissions(examID uint, limit, offset int, includeStats bool, search, orderBy, orderDir string, moodleSynced *bool) (*ListSubmissionsResult, error) {
	var subs []models.UserExamSubmission
	query := s.db.Preload("User").Preload("Answers").
		Where("exam_id = ?", examID).
		Order("submitted_at DESC")
	query = query.Joins("LEFT JOIN exam_user ON exam_user.id = user_exam_submission.user_id")
	if moodleSynced != nil {
		if *moodleSynced {
			query = query.Where("exam_user.moodle_id IS NOT NULL")
		} else {
			query = query.Where("exam_user.moodle_id IS NULL")
		}
	}
	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
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
	if err := query.Find(&subs).Error; err != nil {
		return nil, err
	}

	result := &ListSubmissionsResult{
		Submissions: subs,
	}
	if includeStats {
		var totalCount int64
		if err := s.db.Model(&models.UserExamSubmission{}).
			Where("exam_id = ?", examID).
			Count(&totalCount).Error; err != nil {
			return nil, err
		}

		var avg sql.NullFloat64
		if err := s.db.Model(&models.UserExamSubmission{}).
			Select("AVG(score)").
			Where("exam_id = ? AND score IS NOT NULL", examID).
			Row().Scan(&avg); err != nil {
			return nil, err
		}

		result.TotalSubmissions = totalCount
		if avg.Valid {
			result.AverageScore = &avg.Float64
		}
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

func buildOfficialResultsOrder(orderBy, orderDir string) string {
	order := strings.ToLower(strings.TrimSpace(orderBy))
	dir := strings.ToUpper(strings.TrimSpace(orderDir))
	if dir != "ASC" && dir != "DESC" {
		dir = ""
	}

	switch order {
	case "dni":
		if dir == "" {
			dir = "ASC"
		}
		return fmt.Sprintf("exam_official_result.dni_masked %s", dir)
	case "nombre":
		if dir == "" {
			dir = "ASC"
		}
		return fmt.Sprintf("exam_official_result.nombre %s", dir)
	case "apellidos":
		if dir == "" {
			dir = "ASC"
		}
		return fmt.Sprintf("exam_official_result.apellido_1 %s, exam_official_result.apellido_2 %s", dir, dir)
	case "usuario":
		if dir == "" {
			dir = "ASC"
		}
		return fmt.Sprintf("exam_user.name %s, exam_user.surname %s", dir, dir)
	case "creado", "created_at":
		if dir == "" {
			dir = "DESC"
		}
		return fmt.Sprintf("exam_official_result.created_at %s", dir)
	default:
		if dir == "" {
			dir = "DESC"
		}
		return fmt.Sprintf("exam_official_result.created_at %s", dir)
	}
}

func (s *Service) ListOfficialResults(examID uint, limit, offset int, orderBy, orderDir string) (*OfficialResultsList, error) {
	var (
		results []models.ExamOfficialResult
		total   int64
	)

	query := s.db.Model(&models.ExamOfficialResult{}).
		Where("exam_id = ?", examID)

	if strings.EqualFold(orderBy, "usuario") {
		query = query.Joins("LEFT JOIN exam_user ON exam_user.id = exam_official_result.user_id")
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	orderClause := buildOfficialResultsOrder(orderBy, orderDir)
	if orderClause != "" {
		query = query.Order(orderClause)
	}
	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	if err := query.Preload("User").Find(&results).Error; err != nil {
		return nil, err
	}

	return &OfficialResultsList{
		Results: results,
		Total:   total,
	}, nil
}

func (s *Service) CreateOfficialResult(examID uint, req CreateOfficialResultRequest) (*models.ExamOfficialResult, error) {
	dni := strings.ToUpper(strings.TrimSpace(req.DNI))
	apellido1 := strings.ToUpper(strings.TrimSpace(req.Apellido1))
	nombre := strings.ToUpper(strings.TrimSpace(req.Nombre))

	if dni == "" || apellido1 == "" || nombre == "" {
		return nil, fmt.Errorf("%w: DNI, apellido 1 y nombre son obligatorios", ErrInvalidOfficialResult)
	}

	apellido2Raw := strings.ToUpper(strings.TrimSpace(req.Apellido2))
	var apellido2 *string
	if apellido2Raw != "" {
		apellido2 = &apellido2Raw
	}

	if err := s.db.First(&models.Exam{}, examID).Error; err != nil {
		return nil, err
	}

	var existing models.ExamOfficialResult
	err := s.db.
		Where("exam_id = ? AND dni_masked = ?", examID, dni).
		First(&existing).Error
	if err == nil {
		return nil, ErrOfficialResultExists
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	newResult := models.ExamOfficialResult{
		ExamID:    examID,
		DniMasked: dni,
		Apellido1: apellido1,
		Apellido2: apellido2,
		Nombre:    nombre,
	}

	var user models.ExamUser
	if err := s.db.Where("UPPER(dni) = ?", dni).First(&user).Error; err == nil {
		newResult.UserID = &user.ID
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	if err := s.db.Create(&newResult).Error; err != nil {
		return nil, err
	}

	if err := s.db.Preload("User").First(&newResult, newResult.ID).Error; err != nil {
		return nil, err
	}

	return &newResult, nil
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
