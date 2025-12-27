package admin

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"gorm.io/gorm"

	"github.com/inscripcion-moodle/go-backend/internal/models"
	"github.com/inscripcion-moodle/go-backend/internal/repository"
)

var (
	ErrExamNameConflict      = errors.New("ya existe un examen con ese nombre")
	ErrActiveQuestions       = errors.New("el examen debe tener al menos una pregunta activa no anulada")
	ErrQuestionNotFound      = errors.New("la pregunta no pertenece al examen")
	ErrOfficialResultExists  = errors.New("ya existe un resultado oficial con ese DNI para este examen")
	ErrInvalidOfficialResult = errors.New("datos de resultado oficial invalidos")
	ErrOfficialResultNotFound = errors.New("resultado oficial no encontrado")
	ErrExamNotFound          = errors.New("el examen no existe")
	ErrExamNoQuestions       = errors.New("el examen debe tener al menos una pregunta")
	ErrInvalidOption         = errors.New("opcion de respuesta no valida")
)

type Service struct {
	db             *gorm.DB
	examRepo       repository.ExamRepository
	submissionRepo repository.SubmissionRepository
	officialRepo   repository.OfficialResultRepository
	questionRepo   repository.QuestionRepository
}

func New(db *gorm.DB, examRepo repository.ExamRepository, subRepo repository.SubmissionRepository, offRepo repository.OfficialResultRepository, questionRepo repository.QuestionRepository) *Service {
	return &Service{
		db:             db,
		examRepo:       examRepo,
		submissionRepo: subRepo,
		officialRepo:   offRepo,
		questionRepo:   questionRepo,
	}
}

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

	if err := s.examRepo.CreateExam(context.Background(), s.db, exam); err != nil {
		return nil, err
	}
	return exam, nil
}

func createQuestions(inputs []QuestionInput) ([]models.Question, error) {
	nameGen := newQuestionNameGenerator(nil)
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

	questions, err := s.updateQuestions(exam.ID, exam.Questions, req.Questions)
	if err != nil {
		return nil, err
	}

	if len(activeQuestions(questions)) == 0 {
		return nil, ErrActiveQuestions
	}

	exam.Questions = questions
	if err := s.examRepo.UpdateExam(context.Background(), s.db, exam); err != nil {
		return nil, err
	}

	return exam, nil
}

func (s *Service) updateQuestions(examID uint, existing []models.Question, inputs []QuestionInput) ([]models.Question, error) {
	nameGen := newQuestionNameGenerator(questionNames(existing))
	return s.mergeQuestions(examID, existing, inputs, nameGen)
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
	return s.submissionRepo.FindByID(context.Background(), s.db, submissionID)
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
	answerMap := make(map[uint]*models.UserAnswer)
	for i := range submission.Answers {
		answer := submission.Answers[i]
		answerMap[answer.QuestionID] = &submission.Answers[i]
	}

	ctx := context.Background()
	for _, answer := range req.Answers {
		if existing, ok := answerMap[answer.QuestionID]; ok {
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

func (s *Service) ListOfficialResults(examID uint, limit, offset int, resultType, orderBy, orderDir string) (*OfficialResultsList, error) {
	orderClause := buildOfficialResultsOrder(orderBy, orderDir)
	results, total, err := s.officialRepo.List(context.Background(), s.db, examID, resultType, offset, limit, orderClause)
	if err != nil {
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
		return nil, ErrInvalidOfficialResult
	}

	apellido2Raw := strings.ToUpper(strings.TrimSpace(req.Apellido2))
	var apellido2 *string
	if apellido2Raw != "" {
		apellido2 = &apellido2Raw
	}

	if err := s.db.First(&models.Exam{}, examID).Error; err != nil {
		return nil, err
	}

	existing, err := s.officialRepo.FindByExamAndDNI(context.Background(), s.db, examID, dni)
	if err == nil && existing != nil {
		return nil, ErrOfficialResultExists
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
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
	
	if req.ResultType != "" {
		newResult.ResultType = req.ResultType
	}

	if err := s.officialRepo.Create(context.Background(), s.db, &newResult); err != nil {
		return nil, err
	}

	// We might need to reload if triggers/db sets fields, or just return what we created. 
	// The repo helper Create likely doesn't preload user. 
	// If needed we can do FindByExamAndDNI again or rely on GORM filling ID. 
	// For consistency let's just return the struct as GORM fills ID.
	return &newResult, nil
}

func (s *Service) UpdateOfficialResult(id uint, req EditOfficialResultRequest) (*models.ExamOfficialResult, error) {
	result, err := s.officialRepo.FindByID(context.Background(), s.db, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrOfficialResultNotFound
		}
		return nil, err
	}

	if req.Apellido1 != nil {
		val := strings.ToUpper(strings.TrimSpace(*req.Apellido1))
		if val == "" {
			return nil, ErrInvalidOfficialResult
		}
		result.Apellido1 = val
	}
	if req.Apellido2 != nil {
		val := strings.ToUpper(strings.TrimSpace(*req.Apellido2))
		if val == "" {
			result.Apellido2 = nil
		} else {
			result.Apellido2 = &val
		}
	}
	if req.Nombre != nil {
		val := strings.ToUpper(strings.TrimSpace(*req.Nombre))
		if val == "" {
			return nil, ErrInvalidOfficialResult
		}
		result.Nombre = val
	}
	if req.ResultType != nil {
		result.ResultType = *req.ResultType
	}

	if err := s.officialRepo.Update(context.Background(), s.db, result); err != nil {
		return nil, err
	}

	return result, nil
}

func (s *Service) DeleteOfficialResult(id uint) error {
	_, err := s.officialRepo.FindByID(context.Background(), s.db, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrOfficialResultNotFound
		}
		return err
	}

	return s.officialRepo.Delete(context.Background(), s.db, id)
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
