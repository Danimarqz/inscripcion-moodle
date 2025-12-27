package exam

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"github.com/inscripcion-moodle/go-backend/internal/helpers"
	"github.com/inscripcion-moodle/go-backend/internal/models"
	"github.com/inscripcion-moodle/go-backend/internal/repository"
)

var (
	dniLetters   = "TRWAGMYFPDXBNJZSQVHLCKE"
	validOptions = map[string]struct{}{
		"A": {},
		"B": {},
		"C": {},
		"D": {},
	}
	ErrDNINotValid           = errors.New("dni o nie invalido")
	ErrAlreadySubmitted      = errors.New("ya has enviado este examen")
	ErrExamNotFound          = errors.New("examen no encontrado")
	ErrExamNoQuestions       = errors.New("el examen no tiene preguntas configuradas")
	ErrExamNoActive          = errors.New("el examen no tiene preguntas activas no anuladas configuradas")
	ErrInvalidAnswer         = errors.New("opcion de respuesta no valida")
	ErrSubmissionNotFound    = errors.New("submission not found")
	ErrExamNotActive         = errors.New("exam is not active or responses not visible")
	ErrResultsNotViewable    = errors.New("los resultados no estan disponibles para este examen")
	ErrOfficialResultMissing = errors.New("este apartado es solo para personas que realizaron el examen oficial. Si no puedes registrar tus resultados y si te presentaste, contacta con nosotros: info.opositatcae@gmail.com")
)

func NormalizeDNI(value string) string {
	return strings.ToUpper(strings.TrimSpace(value))
}


func extractDigits(value string) string {
	var b strings.Builder
	for _, r := range value {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func middleFourDigits(value string) string {
	digits := extractDigits(value)
	if len(digits) < 4 {
		return ""
	}
	start := (len(digits) - 4) / 2
	return digits[start : start+4]
}

func ValidateDNINIE(value string) bool {
	if value == "" {
		return false
	}
	upper := strings.ToUpper(strings.TrimSpace(value))
	if len(upper) != 9 {
		return false
	}
	letter := upper[len(upper)-1:]
	if !helpers.IsLetter(letter) {
		return false
	}
	numericPart := upper[:len(upper)-1]
	if upper[0] == 'X' || upper[0] == 'Y' || upper[0] == 'Z' {
		prefix := map[byte]string{'X': "0", 'Y': "1", 'Z': "2"}
		numericPart = prefix[upper[0]] + upper[1:len(upper)-1]
	}
	if !helpers.IsDigits(numericPart) {
		return false
	}
	expected := string(dniLetters[helpers.ToInt(numericPart)%23])
	return expected == letter
}

func ValidateAnswerOption(option string) (string, error) {
	upper := strings.ToUpper(strings.TrimSpace(option))
	if upper == "" {
		return "", ErrInvalidAnswer
	}
	if _, ok := validOptions[upper]; !ok {
		return "", ErrInvalidAnswer
	}
	return upper, nil
}

type Service struct {
	db           *gorm.DB
	repo         repository.ExamRepository
	contactEmail string
}

func NewService(db *gorm.DB, repo repository.ExamRepository, contactEmail string) *Service {
	return &Service{db: db, repo: repo, contactEmail: contactEmail}
}

// recalculateScoresAsync runs the score and percentile recalculation in a separate
// transaction and context, making it safe to run in a background goroutine.
func (s *Service) recalculateScoresAsync(examID uint) {
	// Use a background context so the request context being canceled doesn't kill the job.
	ctx := context.Background()

	// We can't use the original transaction, so we start a new one.
	tx := s.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		// This will be tricky to log without a logger dependency.
		// For now, printing to stderr is a temporary solution.
		fmt.Printf("ERROR: could not begin transaction for async recalculateScores for exam %d: %v\n", examID, tx.Error)
		return
	}
	// Ensure rollback on panic or early return.
	defer tx.Rollback()

	if err := recalculateScores(tx, examID); err != nil {
		fmt.Printf("ERROR: failed to asynchronously recalculate scores for exam %d: %v\n", examID, err)
		return
	}

	if err := tx.Commit().Error; err != nil {
		fmt.Printf("ERROR: could not commit transaction for async recalculateScores for exam %d: %v\n", examID, err)
	}
}

func (s *Service) ProcessExamSubmission(req SubmitExamRequest) (*SubmissionPayload, error) {
	var payload *SubmissionPayload
	err := s.db.Transaction(func(tx *gorm.DB) error {
		p, err := processSubmission(tx, req, s.contactEmail)
		if err != nil {
			return err
		}
		payload = p
		return nil
	})
	if err != nil {
		return nil, err
	}

	// On success, launch the recalculation in the background.
	go s.recalculateScoresAsync(req.ExamID)

	return payload, nil
}

func (s *Service) GetQuestionStubs(ctx context.Context, examID uint) ([]QuestionStub, error) {
	_, err := s.repo.FindExamByID(ctx, s.db, examID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrExamNotFound
		}
		return nil, err
	}

	questions, err := s.repo.FindQuestionsByExamID(ctx, s.db, examID)
	if err != nil {
		return nil, err
	}

	stubs := make([]QuestionStub, 0, len(questions))
	for _, question := range questions {
		stubs = append(stubs, QuestionStub{
			ID:          question.ID,
			Name:        question.Name,
			IsActive:    question.IsActive,
			IsCancelled: question.IsCancelled,
		})
	}
	return stubs, nil
}

func getOrCreateUser(tx *gorm.DB, req SubmitExamRequest) (*models.ExamUser, error) {
	normalizedEmail := strings.ToLower(strings.TrimSpace(req.Email))
	normalizedDNI := NormalizeDNI(req.DNI)
	trimmedName := strings.TrimSpace(req.Name)
	trimmedSurname := strings.TrimSpace(req.Surname)

	var candidate models.ExamUser
	if err := tx.Where("dni = ?", normalizedDNI).First(&candidate).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			candidate = models.ExamUser{
				Name:             trimmedName,
				Surname:          trimmedSurname,
				Email:            normalizedEmail,
				DNI:              normalizedDNI,
				AcceptsMarketing: req.AcceptsMarketing,
			}
			if err := tx.Create(&candidate).Error; err != nil {
				return nil, err
			}
			return &candidate, nil
		}
		return nil, err
	}

	candidate.Name = trimmedName
	candidate.Surname = trimmedSurname
	candidate.Email = normalizedEmail
	candidate.AcceptsMarketing = req.AcceptsMarketing
	if err := tx.Save(&candidate).Error; err != nil {
		return nil, err
	}
	return &candidate, nil
}

func processSubmission(tx *gorm.DB, req SubmitExamRequest, contactEmail string) (*SubmissionPayload, error) {
	normalizedDNI := NormalizeDNI(req.DNI)
	trimmedName := strings.TrimSpace(req.Name)
	trimmedSurname := strings.TrimSpace(req.Surname)
	if !ValidateDNINIE(normalizedDNI) {
		return nil, ErrDNINotValid
	}

	match, err := CheckOfficialResultMatch(tx, OfficialResultMatchRequest{
		ExamID:  req.ExamID,
		Name:    trimmedName,
		Surname: trimmedSurname,
		DNI:     normalizedDNI,
	})
	if err != nil {
		return nil, err
	}
	if !match {
		return nil, fmt.Errorf("este apartado es solo para personas que realizaron el examen oficial. Si no puedes registrar tus resultados y si te presentaste, contacta con nosotros: %s: %w", contactEmail, ErrOfficialResultMissing)
	}

	candidate, err := getOrCreateUser(tx, req)
	if err != nil {
		return nil, err
	}

	var existingCount int64
	if err := tx.Model(&models.UserExamSubmission{}).
		Where("user_id = ? AND exam_id = ?", candidate.ID, req.ExamID).
		Count(&existingCount).Error; err != nil {
		return nil, err
	}
	if existingCount > 0 {
		return nil, errors.New("ya has enviado este examen")
	}

	submission, breakdown, _, err := createSubmission(tx, req, candidate.ID)
	if err != nil {
		return nil, err
	}

	// The score recalculation is now handled asynchronously by the caller.

	if err := tx.Preload("Answers").Preload("User").First(submission, submission.ID).Error; err != nil {
		return nil, err
	}

	payload, err := buildSubmissionPayload(tx, &submission.Exam, submission, "Examen enviado correctamente", breakdown)
	if err != nil {
		return nil, err
	}

	return payload, nil
}

func createSubmission(tx *gorm.DB, req SubmitExamRequest, userID uint) (*models.UserExamSubmission, *ScoreBreakdown, map[uint]string, error) {

	var exam models.Exam

	if err := tx.Preload("Questions").First(&exam, req.ExamID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, nil, ErrExamNotFound
		}
		return nil, nil, nil, err
	}

	if len(exam.Questions) == 0 {
		return nil, nil, nil, ErrExamNoQuestions
	}

	activeQuestions := make([]models.Question, 0, len(exam.Questions))
	activeMap := make(map[uint]models.Question)

	for _, question := range exam.Questions {
		if question.IsActive && !question.IsCancelled {
			activeQuestions = append(activeQuestions, question)
			activeMap[question.ID] = question
		}
	}

	if len(activeQuestions) == 0 {
		return nil, nil, nil, ErrExamNoActive
	}

	answersMap := make(map[uint]string)
	for _, ans := range req.Answers {
		question, ok := activeMap[ans.QuestionID]
		if !ok {
			return nil, nil, nil, fmt.Errorf("pregunta %d no encontrada", ans.QuestionID)
		}

		value, err := ValidateAnswerOption(ans.Answer)
		if err != nil {
			return nil, nil, nil, err
		}
		answersMap[question.ID] = value
	}

	breakdown, err := CalculateScoreBreakdown(activeQuestions, answersMap)
	if err != nil {
		return nil, nil, nil, err
	}

	scorePtr := helpers.Ptr(breakdown.Score)

	submission := &models.UserExamSubmission{
		UserID:             userID,
		ExamID:             req.ExamID,
		Score:              scorePtr,
		Merits:             req.Merits,
		Percentile:         helpers.Ptr(0.0),
		SelectedResultType: req.ResultType,
	}

	if err := tx.Create(submission).Error; err != nil {
		return nil, nil, nil, err
	}

	for _, ans := range req.Answers {
		option, ok := answersMap[ans.QuestionID]
		if !ok {
			continue
		}
		answer := models.UserAnswer{
			SubmissionID: submission.ID,
			QuestionID:   ans.QuestionID,
			Answer:       option,
		}
		if err := tx.Create(&answer).Error; err != nil {
			return nil, nil, nil, err
		}
	}
	return submission, &breakdown, answersMap, nil
}

func BuildSubmissionCheckResponse(db *gorm.DB, req SubmissionCheckRequest) (*SubmissionPayload, error) {
	normalizedEmail := strings.ToLower(strings.TrimSpace(req.Email))
	normalizedDNI := NormalizeDNI(req.DNI)
	var submission models.UserExamSubmission
	queryDB := db.Session(&gorm.Session{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
	if err := queryDB.
		Preload("User").
		Preload("Exam.Questions", "is_active and not is_cancelled").
		Preload("Answers").
		Joins("JOIN exam_user ON exam_user.id = user_exam_submission.user_id").
		Where("exam_user.dni = ? AND LOWER(exam_user.email) = ? AND user_exam_submission.exam_id = ?", normalizedDNI, normalizedEmail, req.ExamID).
		First(&submission).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrSubmissionNotFound
		}
		return nil, err
	}

	if !submission.Exam.IsActive {
		return nil, ErrExamNotActive
	}

	if !submission.Exam.ValidatedTribunal &&
		!submission.Exam.ShowScore &&
		!submission.Exam.ShowPercentile &&
		!submission.Exam.ShowScoreFull {
		return nil, ErrResultsNotViewable
	}

	payload, err := buildSubmissionPayload(db, &submission.Exam, &submission, "Ya has enviado este examen anteriormente", nil)
	if err != nil {
		return nil, err
	}

	return payload, nil
}

func CheckOfficialResultMatch(db *gorm.DB, req OfficialResultMatchRequest) (bool, error) {
	if req.ExamID == 0 {
		return false, ErrExamNotFound
	}

	name := helpers.NormalizeName(req.Name)
	surname := helpers.NormalizeName(req.Surname)
	if name == "" || surname == "" {
		return false, nil
	}

	centerDigits := middleFourDigits(req.DNI)
	if centerDigits == "" {
		return false, nil
	}

	// Buscamos registros que contengan los 4 dígitos centrales en el campo dni_masked.
	pattern := "%" + centerDigits + "%"

	var results []models.ExamOfficialResult
	if err := db.Where("exam_id = ? AND dni_masked LIKE ?", req.ExamID, pattern).Find(&results).Error; err != nil {
		return false, err
	}

	for _, res := range results {
		resName := helpers.NormalizeName(res.Nombre)
		surnameParts := []string{res.Apellido1}
		if res.Apellido2 != nil {
			surnameParts = append(surnameParts, *res.Apellido2)
		}
		resSurname := helpers.NormalizeName(strings.TrimSpace(strings.Join(surnameParts, " ")))
		if resName == "" || resSurname == "" {
			continue
		}
		if resName != name {
			continue
		}
		if resSurname != surname {
			continue
		}
		// Doble check por seguridad, aunque el LIKE ya debería haber filtrado la mayoría
		if middleFourDigits(res.DniMasked) == centerDigits {
			return true, nil
		}
	}

	return false, nil
}

func CalculateScoreBreakdown(questions []models.Question, answers map[uint]string) (ScoreBreakdown, error) {
	total := 0
	correct := 0
	for _, question := range questions {
		if !question.IsActive || question.IsCancelled {
			continue
		}
		total++
		selected := strings.ToUpper(strings.TrimSpace(answers[question.ID]))
		if selected != "" && strings.EqualFold(selected, question.CorrectOption) {
			correct++
		}
	}
	if total == 0 {
		return ScoreBreakdown{}, errors.New("el examen no tiene preguntas activas no anuladas configuradas")
	}
	score := float64(correct) / float64(total) * 100
	return ScoreBreakdown{
		Score:          score,
		CorrectAnswers: correct,
		TotalQuestions: total,
	}, nil
}

func buildSubmissionPayload(tx *gorm.DB, exam *models.Exam, submission *models.UserExamSubmission, message string, breakdown *ScoreBreakdown) (*SubmissionPayload, error) {
	payload := &SubmissionPayload{
		Message: message,
		Merits:  submission.Merits,
	}

	if exam.ShowScore {
		payload.Score = submission.Score
	}

	if exam.ShowPercentile {
		payload.Percentile = submission.Percentile
		position, total, err := getSubmissionPositionData(tx, submission)
		if err != nil {
			return nil, err
		}
		payload.Position = position
		payload.TotalSubmissions = total
	}

	if exam.ShowScoreFull {
		var breakdownData *ScoreBreakdown
		if breakdown != nil {
			breakdownCopy := *breakdown
			breakdownData = &breakdownCopy
		} else {
			data, err := fetchScoreBreakdownFromDB(tx, exam.ID, submission.ID)
			if err != nil {
				return nil, err
			}
			breakdownData = data
		}

		if breakdownData != nil {
			payload.CorrectAnswers = helpers.Ptr(breakdownData.CorrectAnswers)
			payload.TotalQuestions = helpers.Ptr(breakdownData.TotalQuestions)
		}
	}

	if exam.ValidatedTribunal {
		review, err := buildAnswersReview(tx, exam, submission)
		if err != nil {
			return nil, err
		}
		payload.AnswersReview = review
	}

	return payload, nil
}

func getSubmissionPositionData(tx *gorm.DB, submission *models.UserExamSubmission) (*int, *int, error) {
	var total int64
	if err := tx.Model(&models.UserExamSubmission{}).
		Where("exam_id = ?", submission.ExamID).
		Count(&total).Error; err != nil {
		return nil, nil, err
	}
	totalSubmissions := int(total)
	if totalSubmissions == 0 || submission.Score == nil {
		return nil, helpers.Ptr(totalSubmissions), nil
	}

	var better int64
	if err := tx.Model(&models.UserExamSubmission{}).
		Where("exam_id = ? AND score IS NOT NULL AND score > ?", submission.ExamID, *submission.Score).
		Count(&better).Error; err != nil {
		return nil, nil, err
	}
	position := int(better) + 1
	return helpers.Ptr(position), helpers.Ptr(totalSubmissions), nil
}

func fetchScoreBreakdownFromDB(tx *gorm.DB, examID, submissionID uint) (*ScoreBreakdown, error) {
	var questions []models.Question
	if err := tx.Where("exam_id = ? AND is_active = 1 AND is_cancelled = 0", examID).Find(&questions).Error; err != nil {
		return nil, err
	}
	if len(questions) == 0 {
		return nil, nil
	}

	var answers []models.UserAnswer
	if err := tx.Where("submission_id = ?", submissionID).Find(&answers).Error; err != nil {
		return nil, err
	}

	answerMap := make(map[uint]string)
	for _, ans := range answers {
		answerMap[ans.QuestionID] = ans.Answer
	}

	breakdown, err := CalculateScoreBreakdown(questions, answerMap)
	if err != nil {
		return nil, err
	}
	return &breakdown, nil
}

func buildAnswersReview(tx *gorm.DB, exam *models.Exam, submission *models.UserExamSubmission) ([]AnswerReview, error) {
	questions := exam.Questions
	if len(questions) == 0 {
		if err := tx.Where("exam_id = ?", exam.ID).Find(&questions).Error; err != nil {
			return nil, err
		}
	}

	answers := submission.Answers
	if len(answers) == 0 {
		if err := tx.Where("submission_id = ?", submission.ID).Find(&answers).Error; err != nil {
			return nil, err
		}
	}

	answerMap := make(map[uint]string)
	for _, ans := range answers {
		answerMap[ans.QuestionID] = strings.ToUpper(ans.Answer)
	}

	sort.SliceStable(questions, func(i, j int) bool {
		if questions[i].Name == questions[j].Name {
			return questions[i].ID < questions[j].ID
		}
		return questions[i].Name < questions[j].Name
	})

	review := make([]AnswerReview, 0, len(questions))
	for _, question := range questions {
		if !question.IsActive || question.IsCancelled {
			continue
		}
		selected, has := answerMap[question.ID]
		var selectedPtr *string
		if has {
			selectedPtr = helpers.Ptr(selected)
		}
		var correctPtr *string
		if question.CorrectOption != "" {
			correctPtr = helpers.Ptr(strings.ToUpper(question.CorrectOption))
		}
		review = append(review, AnswerReview{
			QuestionID:     question.ID,
			QuestionLabel:  helpers.Ptr(question.Name),
			SelectedOption: selectedPtr,
			CorrectOption:  correctPtr,
			IsCorrect:      has && correctPtr != nil && selectedPtr != nil && strings.EqualFold(*selectedPtr, *correctPtr),
		})
	}

	return review, nil
}

func recalculateScores(tx *gorm.DB, examID uint) error {
	const scoreSQL = `
UPDATE user_exam_submission AS u
JOIN (
    SELECT ua.submission_id,
           ROUND(SUM(CASE WHEN UPPER(ua.answer) = q.correct_option THEN 1 ELSE 0 END) / COUNT(q.id) * 100, 2) AS base_score
    FROM user_answer ua
    JOIN question q ON q.id = ua.question_id
    WHERE q.exam_id = ?
      AND q.is_active = 1
      AND NOT q.is_cancelled
    GROUP BY ua.submission_id
) AS t ON u.id = t.submission_id
SET u.score = t.base_score
WHERE u.exam_id = ?`
	if err := tx.Exec(scoreSQL, examID, examID).Error; err != nil {
		return err
	}

	const percentileSQL = `
UPDATE user_exam_submission AS u
JOIN (
    SELECT
        id,
        exam_id,
        ROUND(CUME_DIST() OVER (PARTITION BY exam_id ORDER BY score) * 100, 2) AS pct
    FROM user_exam_submission
    WHERE exam_id = ? AND score IS NOT NULL
) ranked ON ranked.id = u.id
SET u.percentile = ranked.pct
WHERE u.exam_id = ?`
	return tx.Exec(percentileSQL, examID, examID).Error
}
