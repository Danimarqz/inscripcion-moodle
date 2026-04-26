package exam

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math"
	"slices"
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
	ErrOfficialResultMissing = errors.New("este apartado es solo para personas que realizaron el examen oficial. Si no puedes registrar tus resultados y si te presentaste, contacta con nosotros: info@opositatcae.es")
	ErrMeritsAlreadySet      = errors.New("los méritos ya han sido guardados y no se pueden modificar")
)

func NormalizeDNI(value string) string {
	return strings.ToUpper(strings.TrimSpace(value))
}

func CreateSlug(value string) string {
	return helpers.CreateSlug(value)
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
	value = strings.TrimSpace(value)
	if len(value) == 0 {
		return ""
	}

	digits := extractDigits(value)

	if len(digits) == 4 {
		return digits
	}

	if value[0] >= '0' && value[0] <= '9' {
		// DNI LOGIC
		if len(digits) >= 7 {
			return digits[3:7]
		}
	} else {
		// NIE LOGIC
		if len(digits) >= 6 {
			return digits[2:6]
		}
	}
	return ""
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
	db             *gorm.DB
	repo           repository.ExamRepository
	submissionRepo repository.SubmissionRepository
	contactEmail   string
}

func NewService(db *gorm.DB, repo repository.ExamRepository, submissionRepo repository.SubmissionRepository, contactEmail string) *Service {
	return &Service{db: db, repo: repo, submissionRepo: submissionRepo, contactEmail: contactEmail}
}

// recalculatePercentilesAsync runs the percentile recalculation in a separate
// transaction and context, making it safe to run in a background goroutine.
// Scores are already calculated upon submission, so we only need to update percentiles.
func (s *Service) recalculatePercentilesAsync(examID uint) {
	// Use a background context so the request context being canceled doesn't kill the job.
	ctx := context.Background()

	// We can't use the original transaction, so we start a new one.
	tx := s.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		// This will be tricky to log without a logger dependency.
		// For now, printing to stderr is a temporary solution.
		log.Printf("ERROR: could not begin transaction for async recalculatePercentiles for exam %d: %v", examID, tx.Error)
		return
	}
	// Ensure rollback on panic or early return.
	defer tx.Rollback()

	if err := s.repo.RecalculatePercentiles(ctx, tx, examID); err != nil {
		log.Printf("ERROR: failed to asynchronously recalculate percentiles for exam %d: %v", examID, err)
		return
	}

	if err := tx.Commit().Error; err != nil {
		log.Printf("ERROR: could not commit transaction for async recalculatePercentiles for exam %d: %v", examID, err)
	}
}

func (s *Service) ProcessExamSubmission(req SubmitExamRequest) (*SubmissionPayload, error) {
	var payload *SubmissionPayload
	err := s.db.Transaction(func(tx *gorm.DB) error {
		p, err := s.processSubmission(tx, req, s.contactEmail, s.repo, s.submissionRepo)
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
	go s.recalculatePercentilesAsync(req.ExamID)

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
			Label:       question.Label,
			IsActive:    question.IsActive,
			IsCancelled: question.IsCancelled,
		})
	}
	return stubs, nil
}

func (s *Service) GetExamBySlug(ctx context.Context, slug string) (*models.Exam, error) {
	exam, err := s.repo.FindExamBySlug(ctx, s.db, slug)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrExamNotFound
		}
		return nil, err
	}
	return exam, nil
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

func (s *Service) processSubmission(tx *gorm.DB, req SubmitExamRequest, contactEmail string, examRepo repository.ExamRepository, submissionRepo repository.SubmissionRepository) (*SubmissionPayload, error) {
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

	if err := tx.Preload("User").First(submission, submission.ID).Error; err != nil {
		return nil, err
	}

	// Determine pass status message
	message := "Examen enviado correctamente"
	threshold := submission.Exam.PassingThreshold
	isPassed := evaluatePassStatus(submission.Score, threshold)
	if isPassed != nil && *isPassed {
		message = "Enhorabuena, hay posibilidades de que pases el corte. El siguiente paso es meter tus méritos, cuando los tengas calculados añádelos"
	}

	payload, err := s.buildSubmissionPayload(tx, &submission.Exam, submission, message, breakdown, examRepo, submissionRepo)
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
	questionsMap := make(map[uint]models.Question)

	for _, question := range exam.Questions {
		questionsMap[question.ID] = question
		if question.IsActive && !question.IsCancelled {
			activeQuestions = append(activeQuestions, question)
		}
	}

	if len(activeQuestions) == 0 {
		return nil, nil, nil, ErrExamNoActive
	}

	answersMap := make(map[uint]string)
	for _, ans := range req.Answers {
		question, ok := questionsMap[ans.QuestionID]
		if !ok {
			return nil, nil, nil, fmt.Errorf("pregunta %d no encontrada", ans.QuestionID)
		}

		value, err := ValidateAnswerOption(ans.Answer)
		if err != nil {
			return nil, nil, nil, err
		}
		answersMap[question.ID] = value
	}

	penalty := 0.0
	if exam.PenaltyValue != nil {
		penalty = *exam.PenaltyValue
	}
	maxScore := 100.0
	if exam.MaxScore != nil {
		maxScore = *exam.MaxScore
	}
	breakdown, err := CalculateScoreBreakdown(activeQuestions, answersMap, exam.SubtractsPoints, penalty, maxScore)
	if err != nil {
		return nil, nil, nil, err
	}

	scorePtr := new(breakdown.Score)
	answersJSON := models.AnswersJSON(answersMap)

	submission := &models.UserExamSubmission{
		UserID:             userID,
		ExamID:             req.ExamID,
		Score:              scorePtr,
		Merits:             req.Merits,
		Percentile:         new(0.0),
		SelectedResultType: req.ResultType,
		AnswersData:        &answersJSON,
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
	return submission, breakdown, answersMap, nil
}

func (s *Service) BuildSubmissionCheckResponse(req SubmissionCheckRequest) (*SubmissionPayload, error) {
	normalizedEmail := strings.ToLower(strings.TrimSpace(req.Email))
	normalizedDNI := NormalizeDNI(req.DNI)
	var submission models.UserExamSubmission
	queryDB := s.db.Session(&gorm.Session{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
	if err := queryDB.
		Preload("User").
		Preload("Exam.Questions", "is_active and not is_cancelled").
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

	payload, err := s.buildSubmissionPayload(s.db, &submission.Exam, &submission, "Ya has enviado este examen anteriormente", nil, s.repo, s.submissionRepo)
	if err != nil {
		return nil, err
	}

	return payload, nil
}

var (
	ErrNotPassed     = errors.New("no has aprobado el examen, no puedes añadir méritos")
	ErrMeritsInvalid = errors.New("el valor de méritos no es válido")
)

func (s *Service) UpdateMerits(req UpdateMeritsRequest) (*UpdateMeritsResponse, error) {
	normalizedDNI := NormalizeDNI(req.DNI)
	if !ValidateDNINIE(normalizedDNI) {
		return nil, ErrDNINotValid
	}

	normalizedEmail := strings.ToLower(strings.TrimSpace(req.Email))

	var submission models.UserExamSubmission
	queryDB := s.db.Session(&gorm.Session{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
	if err := queryDB.
		Preload("Exam").
		Joins("JOIN exam_user ON exam_user.id = user_exam_submission.user_id").
		Where("exam_user.dni = ? AND LOWER(exam_user.email) = ? AND user_exam_submission.exam_id = ?", normalizedDNI, normalizedEmail, req.ExamID).
		First(&submission).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrSubmissionNotFound
		}
		return nil, err
	}

	if submission.Merits != nil {
		return nil, ErrMeritsAlreadySet
	}

	// Use official score for pass evaluation when override is enabled
	effectiveScore := submission.Score
	if submission.Exam.UseOfficialScores && submission.UserID != 0 {
		var official models.ExamOfficialResult
		if err := s.db.Where("exam_id = ? AND user_id = ?", submission.Exam.ID, submission.UserID).First(&official).Error; err == nil {
			if official.Score != nil {
				effectiveScore = official.Score
			}
		}
	}

	threshold := submission.Exam.PassingThreshold
	isPassed := evaluatePassStatus(effectiveScore, threshold)
	if isPassed == nil || !*isPassed {
		return nil, ErrNotPassed
	}

	if req.Merits != nil && *req.Merits > submission.Exam.MaxMerits {
		return nil, fmt.Errorf("el valor de méritos no puede superar %.2f", submission.Exam.MaxMerits)
	}

	submission.Merits = req.Merits
	if err := s.db.Model(&submission).Update("merits", req.Merits).Error; err != nil {
		return nil, err
	}

	resp := &UpdateMeritsResponse{
		Message: "Méritos actualizados correctamente",
		Merits:  req.Merits,
	}

	if req.Merits != nil && threshold != nil {
		pos, total, err := s.submissionRepo.GetMeritsRanking(context.Background(), s.db, req.ExamID, submission.ID, *threshold, submission.Exam.ExamWeight, submission.Exam.SkipWeights)
		if err == nil {
			resp.MeritsPosition = pos
			resp.MeritsTotal = total
		}
	}

	if req.Merits != nil && effectiveScore != nil {
		var ws float64
		if submission.Exam.SkipWeights {
			ws = math.Round((*effectiveScore+*req.Merits)*1000) / 1000
		} else {
			ew := submission.Exam.ExamWeight
			ws = math.Round((*effectiveScore*ew+*req.Merits*(1-ew))*1000) / 1000
		}
		resp.WeightedScore = &ws
	}

	return resp, nil
}

// anyWordMatches returns true if any word in userValue matches any word in officialValue.
// Both values must be already normalized (uppercased, no accents).
func anyWordMatches(userValue, officialValue string) bool {
	userWords := strings.Fields(userValue)
	officialWords := strings.Fields(officialValue)
	for _, uw := range userWords {
		if slices.Contains(officialWords, uw) {
			return true
		}
	}
	return false
}

// anySurnameWordMatches returns true if any word in the user's surname matches
// either apellido1 or apellido2 from the official result.
// Both apellidos are already normalized individually.
func anySurnameWordMatches(userSurname, apellido1, apellido2 string) bool {
	userWords := strings.FieldsSeq(userSurname)
	for uw := range userWords {
		if slices.Contains(strings.Fields(apellido1), uw) {
			return true
		}
		if apellido2 != "" {
			if slices.Contains(strings.Fields(apellido2), uw) {
				return true
			}
		}
	}
	return false
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
		resApellido1 := helpers.NormalizeName(res.Apellido1)
		var resApellido2 string
		if res.Apellido2 != nil {
			resApellido2 = helpers.NormalizeName(*res.Apellido2)
		}
		if resName == "" || resApellido1 == "" {
			continue
		}

		// Comprobación flexible de nombre: si alguna palabra del usuario
		// coincide con alguna palabra del nombre oficial, es suficiente.
		if !anyWordMatches(name, resName) {
			continue
		}

		// Comprobación flexible de apellidos: si alguna palabra del usuario
		// coincide con Apellido1 o Apellido2, es suficiente.
		if !anySurnameWordMatches(surname, resApellido1, resApellido2) {
			continue
		}

		// Doble check por seguridad, aunque el LIKE ya debería haber filtrado la mayoría
		if middleFourDigits(res.DniMasked) == centerDigits {
			return true, nil
		}
	}

	return false, nil
}

func CalculateScoreBreakdown(questions []models.Question, answers map[uint]string, subtracts bool, penalty float64, maxScore float64) (*ScoreBreakdown, error) {
	total := 0
	correct := 0
	incorrect := 0
	for _, question := range questions {
		if !question.IsActive || question.IsCancelled {
			continue
		}
		total++
		qID := question.ID
		selected, ok := answers[qID]
		if !ok {
			continue
		}

		selected = strings.ToUpper(strings.TrimSpace(selected))
		if selected != "" {
			if strings.EqualFold(selected, strings.TrimSpace(question.CorrectOption)) {
				correct++
			} else {
				incorrect++
			}
		}
	}
	if total == 0 {
		return nil, errors.New("el examen no tiene preguntas activas no anuladas configuradas")
	}

	netCorrect := float64(correct)
	if subtracts && penalty > 0 {
		netCorrect -= float64(incorrect) * penalty
	}

	score := netCorrect / float64(total) * maxScore

	notAnswered := total - correct - incorrect

	return &ScoreBreakdown{
		Score:            score,
		CorrectAnswers:   correct,
		IncorrectAnswers: incorrect,
		NotAnswered:      notAnswered,
		TotalQuestions:   total,
	}, nil
}

// ComputePassingThreshold calculates the threshold from criteria and stores it
// in exam.PassingThreshold. Called once when the admin edits the exam.
func ComputePassingThreshold(db *gorm.DB, exam *models.Exam, examRepo repository.ExamRepository) *float64 {
	switch exam.PassingCriteriaType {
	case "min_score":
		return exam.PassingCriteriaValue
	case "top10_pct":
		if exam.PassingCriteriaValue == nil {
			return nil
		}
		avg, err := examRepo.GetTop10AverageScore(context.Background(), db, exam.ID)
		if err != nil || avg == nil {
			return nil
		}
		threshold := (*exam.PassingCriteriaValue / 100) * *avg
		return &threshold
	default:
		return nil
	}
}

func evaluatePassStatus(score *float64, threshold *float64) *bool {
	if threshold == nil {
		return nil
	}
	if score == nil {
		passed := false
		return &passed
	}
	passed := *score >= *threshold
	return &passed
}

// applyOfficialScoreOverride overrides submission score/merits in memory when
// the exam has UseOfficialScores enabled and a linked official result exists.
func applyOfficialScoreOverride(db *gorm.DB, exam *models.Exam, submission *models.UserExamSubmission) {
	if !exam.UseOfficialScores || submission.UserID == 0 {
		return
	}
	var official models.ExamOfficialResult
	if err := db.Where("exam_id = ? AND user_id = ?", exam.ID, submission.UserID).First(&official).Error; err != nil {
		return
	}
	if official.Score != nil {
		v := *official.Score
		submission.Score = &v
	}
	if official.Merits != nil {
		v := *official.Merits
		submission.Merits = &v
	}
}

func (s *Service) buildSubmissionPayload(tx *gorm.DB, exam *models.Exam, submission *models.UserExamSubmission, message string, breakdown *ScoreBreakdown, examRepo repository.ExamRepository, submissionRepo repository.SubmissionRepository) (*SubmissionPayload, error) {
	applyOfficialScoreOverride(tx, exam, submission)

	payload := &SubmissionPayload{
		Message:            message,
		Merits:             submission.Merits,
		MaxScore:           exam.MaxScore,
		SecondaryMaxScores: exam.SecondaryMaxScores,
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
			data, err := fetchScoreBreakdownFromDB(tx, exam, submission.AnswersData)
			if err != nil {
				return nil, err
			}
			breakdownData = data
		}

		if breakdownData != nil {
			payload.CorrectAnswers = new(breakdownData.CorrectAnswers)
			payload.IncorrectAnswers = new(breakdownData.IncorrectAnswers)
			payload.NotAnswered = new(breakdownData.NotAnswered)
			payload.TotalQuestions = new(breakdownData.TotalQuestions)
		}
	}

	if exam.ValidatedTribunal {
		review, err := buildAnswersReview(tx, exam, submission)
		if err != nil {
			return nil, err
		}
		payload.AnswersReview = review
	}

	// Passing criteria logic
	threshold := exam.PassingThreshold
	isPassed := evaluatePassStatus(submission.Score, threshold)
	payload.IsPassed = isPassed
	payload.CanEditMerits = isPassed != nil && *isPassed
	mm := exam.MaxMerits
	payload.MaxMerits = &mm

	if threshold != nil {
		var passedCount int64
		tx.Model(&models.UserExamSubmission{}).
			Where("exam_id = ? AND score >= ?", exam.ID, *threshold).
			Count(&passedCount)
		pc := int(passedCount)
		payload.PassedCount = &pc
	}

	if payload.CanEditMerits && submission.Merits != nil {
		pos, total, err := submissionRepo.GetMeritsRanking(context.Background(), tx, exam.ID, submission.ID, *threshold, exam.ExamWeight, exam.SkipWeights)
		if err == nil {
			payload.MeritsPosition = pos
			payload.MeritsTotal = total
		}
	}

	// Compute weighted score
	if submission.Score != nil && submission.Merits != nil {
		var ws float64
		if exam.SkipWeights {
			ws = math.Round((*submission.Score+*submission.Merits)*1000) / 1000
		} else {
			ws = math.Round((*submission.Score*exam.ExamWeight+*submission.Merits*(1-exam.ExamWeight))*1000) / 1000
		}
		payload.WeightedScore = &ws
	}

	// Send display weights to the student (override if configured)
	displayEW := exam.ExamWeight
	if exam.DisplayExamWeight != nil {
		displayEW = *exam.DisplayExamWeight
	}
	payload.ExamWeight = &displayEW

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
		return nil, new(totalSubmissions), nil
	}

	var better int64
	if err := tx.Model(&models.UserExamSubmission{}).
		Where("exam_id = ? AND score IS NOT NULL AND score > ? AND id != ?", submission.ExamID, *submission.Score, submission.ID).
		Count(&better).Error; err != nil {
		return nil, nil, err
	}
	position := int(better) + 1
	return new(position), new(totalSubmissions), nil
}

func fetchScoreBreakdownFromDB(tx *gorm.DB, exam *models.Exam, answersData *models.AnswersJSON) (*ScoreBreakdown, error) {
	var questions []models.Question
	if err := tx.Where("exam_id = ? AND is_active = 1 AND is_cancelled = 0", exam.ID).Find(&questions).Error; err != nil {
		return nil, err
	}
	if len(questions) == 0 {
		return nil, nil
	}

	answerMap := make(map[uint]string)
	if answersData != nil {
		answerMap = map[uint]string(*answersData)
	}

	penalty := 0.0
	if exam.PenaltyValue != nil {
		penalty = *exam.PenaltyValue
	}
	maxScore := 100.0
	if exam.MaxScore != nil {
		maxScore = *exam.MaxScore
	}
	breakdown, err := CalculateScoreBreakdown(questions, answerMap, exam.SubtractsPoints, penalty, maxScore)
	if err != nil {
		return nil, err
	}
	return breakdown, nil
}

func effectiveQuestionLabel(q models.Question) string {
	prefix := "Pregunta"
	if !q.IsActive {
		prefix = "Reserva"
	}
	if q.Label != nil {
		if trimmed := strings.TrimSpace(*q.Label); trimmed != "" {
			return fmt.Sprintf("%s %s", prefix, trimmed)
		}
	}
	return fmt.Sprintf("%s %d", prefix, q.Name)
}

func buildAnswersReview(tx *gorm.DB, exam *models.Exam, submission *models.UserExamSubmission) ([]AnswerReview, error) {
	questions := exam.Questions
	if len(questions) == 0 {
		if err := tx.Where("exam_id = ?", exam.ID).Find(&questions).Error; err != nil {
			return nil, err
		}
	}

	answerMap := make(map[uint]string)
	if submission.AnswersData != nil {
		for k, v := range *submission.AnswersData {
			answerMap[k] = strings.ToUpper(v)
		}
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
			selectedPtr = new(selected)
		}
		var correctPtr *string
		if question.CorrectOption != "" {
			correctPtr = new(strings.ToUpper(question.CorrectOption))
		}
		review = append(review, AnswerReview{
			QuestionID:     question.ID,
			QuestionLabel:  effectiveQuestionLabel(question),
			SelectedOption: selectedPtr,
			CorrectOption:  correctPtr,
			IsCorrect:      has && correctPtr != nil && selectedPtr != nil && strings.EqualFold(*selectedPtr, *correctPtr),
		})
	}

	return review, nil
}
