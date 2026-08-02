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
	"sync"
	"time"

	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"github.com/inscripcion-moodle/go-backend/internal/helpers"
	"github.com/inscripcion-moodle/go-backend/internal/models"
	"github.com/inscripcion-moodle/go-backend/internal/repository"
	"github.com/inscripcion-moodle/go-backend/internal/scoring"
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
	ErrRaffleNotAccepted     = errors.New("debes leer y aceptar las bases del sorteo")
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
	percentiles    *percentileCoalescer
}

func NewService(db *gorm.DB, repo repository.ExamRepository, submissionRepo repository.SubmissionRepository, contactEmail string) *Service {
	s := &Service{db: db, repo: repo, submissionRepo: submissionRepo, contactEmail: contactEmail}
	// Coalesce percentile recalcs: a burst of submissions to one exam collapses
	// into a single CUME_DIST() pass instead of one full-pool recompute each.
	s.percentiles = newPercentileCoalescer(s.recalculatePercentilesAsync, 2*time.Second)
	return s
}

// percentileCoalescer collapses concurrent/rapid percentile recalcs per exam.
// While a recalc for an exam runs, further triggers set a "pending" flag instead
// of launching another; one follow-up runs after to cover late arrivals. So a
// burst of N submissions yields at most 2 recalcs, not N.
// ponytail: in-process, per-instance. Multi-replica deploys recalc once per
// replica — fine, recalc is idempotent (recomputes from current DB state).
type percentileCoalescer struct {
	mu      sync.Mutex
	pending map[uint]struct{}
	running map[uint]struct{}
	run     func(uint)
	delay   time.Duration
}

func newPercentileCoalescer(run func(uint), delay time.Duration) *percentileCoalescer {
	return &percentileCoalescer{
		pending: make(map[uint]struct{}),
		running: make(map[uint]struct{}),
		run:     run,
		delay:   delay,
	}
}

func (c *percentileCoalescer) trigger(examID uint) {
	c.mu.Lock()
	if _, ok := c.running[examID]; ok {
		c.pending[examID] = struct{}{}
		c.mu.Unlock()
		return
	}
	c.running[examID] = struct{}{}
	c.mu.Unlock()
	go c.loop(examID)
}

func (c *percentileCoalescer) loop(examID uint) {
	for {
		if c.delay > 0 {
			time.Sleep(c.delay) // batch a burst into one recalc
		}
		c.run(examID)
		c.mu.Lock()
		if _, ok := c.pending[examID]; ok {
			delete(c.pending, examID)
			c.mu.Unlock()
			continue // submissions arrived mid-recalc; run once more
		}
		delete(c.running, examID)
		c.mu.Unlock()
		return
	}
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

	// On success, schedule a coalesced background percentile recalculation.
	s.percentiles.trigger(req.ExamID)

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

	official, err := FindOfficialResult(tx, OfficialResultMatchRequest{
		ExamID:  req.ExamID,
		Name:    trimmedName,
		Surname: trimmedSurname,
		DNI:     normalizedDNI,
	})
	if err != nil {
		return nil, err
	}
	if official == nil {
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

	// Si el resultado oficial ya trae nota, el alumno no responde preguntas: la
	// entrega se crea con esa nota. Lo decide el servidor, no el cliente.
	if official.Score != nil {
		return s.processOfficialOnlySubmission(tx, req, official, candidate, examRepo, submissionRepo)
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

// processOfficialOnlySubmission crea la entrega de un alumno cuya fila oficial
// ya trae nota: no hay respuestas que corregir, solo se registran sus datos.
// Repite los controles que hace createSubmission (existencia del examen y
// aceptación del sorteo) porque este camino no pasa por ella.
func (s *Service) processOfficialOnlySubmission(tx *gorm.DB, req SubmitExamRequest, official *models.ExamOfficialResult, user *models.ExamUser, examRepo repository.ExamRepository, submissionRepo repository.SubmissionRepository) (*SubmissionPayload, error) {
	var exam models.Exam
	if err := tx.Preload("Groups").First(&exam, req.ExamID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrExamNotFound
		}
		return nil, err
	}

	if exam.RaffleEnabled && !req.RaffleAccepted {
		return nil, ErrRaffleNotAccepted
	}

	resultType := official.ResultType
	if strings.TrimSpace(resultType) == "" {
		resultType = req.ResultType
	}

	score := *official.Score
	emptyAnswers := models.AnswersJSON{}
	submission := &models.UserExamSubmission{
		UserID:             user.ID,
		ExamID:             req.ExamID,
		Score:              &score,
		Merits:             req.Merits, // los oficiales se aplican en memoria; persistirlos bloquearía UpdateMerits
		Percentile:         new(0.0),
		SelectedResultType: resultType,
		AnswersData:        &emptyAnswers,
	}
	if err := tx.Create(submission).Error; err != nil {
		return nil, err
	}

	// Linkeamos la fila oficial al alumno para que el override y el percentil la
	// encuentren, pero solo si nadie la había linkado ya (el flujo de Moodle
	// puede haberla asignado a otra persona por matching difuso).
	if official.UserID == nil {
		if err := tx.Model(&models.ExamOfficialResult{}).
			Where("id = ? AND user_id IS NULL", official.ID).
			Update("user_id", user.ID).Error; err != nil {
			return nil, err
		}
	}

	submission.User = *user
	submission.Exam = exam

	message := "Hemos registrado tus datos y tu nota oficial"
	isPassed := evaluatePassStatus(submission.Score, exam.PassingThreshold)
	if isPassed != nil && *isPassed {
		message = "Enhorabuena, hay posibilidades de que pases el corte. El siguiente paso es meter tus méritos, cuando los tengas calculados añádelos"
	}

	return s.buildSubmissionPayload(tx, &exam, submission, message, nil, examRepo, submissionRepo)
}

func createSubmission(tx *gorm.DB, req SubmitExamRequest, userID uint) (*models.UserExamSubmission, *ScoreBreakdown, map[uint]string, error) {

	var exam models.Exam

	if err := tx.Preload("Questions").Preload("Groups").First(&exam, req.ExamID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, nil, ErrExamNotFound
		}
		return nil, nil, nil, err
	}

	if exam.RaffleEnabled && !req.RaffleAccepted {
		return nil, nil, nil, ErrRaffleNotAccepted
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

	var breakdown *ScoreBreakdown
	var err error
	if len(exam.Groups) > 0 {
		if exam.ScoringMode == "xunta" {
			breakdown, err = CalculateGroupedXuntaBreakdown(exam.Groups, activeQuestions, answersMap)
		} else {
			breakdown, err = CalculateGroupedBreakdown(exam.Groups, activeQuestions, answersMap, ScoringConfigFromExam(&exam).WrongBlockSize)
		}
	} else {
		breakdown, err = CalculateScoreBreakdownCfg(activeQuestions, answersMap, ScoringConfigFromExam(&exam))
	}
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

	// Answers live in answers_json (set above); the legacy per-row user_answer
	// dual-write was removed — nothing reads it as a source of truth.
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
		Preload("Exam.Groups").
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
		Preload("User").
		Joins("JOIN exam_user ON exam_user.id = user_exam_submission.user_id").
		Where("exam_user.dni = ? AND LOWER(exam_user.email) = ? AND user_exam_submission.exam_id = ?", normalizedDNI, normalizedEmail, req.ExamID).
		First(&submission).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrSubmissionNotFound
		}
		return nil, err
	}

	if submission.Merits != nil && !submission.Exam.AllowMeritsEdit {
		return nil, ErrMeritsAlreadySet
	}

	// Use official score for pass evaluation when an official result exists
	effectiveScore := submission.Score
	if official := findOfficialForSubmission(s.db, &submission.Exam, &submission); official != nil && official.Score != nil {
		effectiveScore = official.Score
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

// FindOfficialResult devuelve la fila oficial que corresponde al alumno, o nil
// si no hay coincidencia. Cuando varias filas casan (p. ej. la misma persona en
// dos convocatorias) se prefiere la que tiene nota y, a igualdad, el id menor,
// para que la elección sea determinista.
func FindOfficialResult(db *gorm.DB, req OfficialResultMatchRequest) (*models.ExamOfficialResult, error) {
	if req.ExamID == 0 {
		return nil, ErrExamNotFound
	}

	name := helpers.NormalizeName(req.Name)
	surname := helpers.NormalizeName(req.Surname)
	if name == "" || surname == "" {
		return nil, nil
	}

	fullDNI := NormalizeDNI(req.DNI)
	if fullDNI == "" {
		return nil, nil
	}

	// Camino rápido: dni_search es una columna generada STORED que guarda solo los
	// dígitos que revela la máscara de cada fila, indexada por (exam_id, dni_search).
	// Los dígitos que revela una máscara coincidente son siempre un tramo contiguo
	// de los dígitos del DNI del usuario, así que buscamos por cada subcadena
	// contigua en lugar de cargar el examen entero. La cadena vacía cubre filas
	// totalmente enmascaradas.
	candidates := dniSearchCandidates(fullDNI)
	var results []models.ExamOfficialResult
	if err := db.Where("exam_id = ? AND dni_search IN ?", req.ExamID, candidates).
		Find(&results).Error; err != nil {
		return nil, err
	}
	if row := pickOfficialRow(results, name, surname, fullDNI); row != nil {
		return row, nil
	}

	// Red de seguridad: una máscara que revelara dígitos NO contiguos no la
	// alcanzaría la búsqueda por subcadenas. Solo cuando el camino rápido no
	// encuentra coincidencia pagamos el escaneo completo, de modo que nunca
	// bloqueamos por error a un alumno por una máscara inusual. El resultado es
	// idéntico al del escaneo original.
	var all []models.ExamOfficialResult
	if err := db.Where("exam_id = ?", req.ExamID).Find(&all).Error; err != nil {
		return nil, err
	}
	return pickOfficialRow(all, name, surname, fullDNI), nil
}

// dniSearchCandidates devuelve cada tramo contiguo de dígitos del DNI completo
// más la cadena vacía, para buscar contra la columna generada dni_search.
func dniSearchCandidates(fullDNI string) []string {
	var digits []byte
	for i := 0; i < len(fullDNI); i++ {
		if fullDNI[i] >= '0' && fullDNI[i] <= '9' {
			digits = append(digits, fullDNI[i])
		}
	}
	set := map[string]struct{}{"": {}}
	for i := 0; i < len(digits); i++ {
		for j := i + 1; j <= len(digits); j++ {
			set[string(digits[i:j])] = struct{}{}
		}
	}
	out := make([]string, 0, len(set))
	for c := range set {
		out = append(out, c)
	}
	return out
}

// pickOfficialRow aplica la comprobación flexible de nombre/apellidos y la
// máscara de DNI sobre las filas dadas y devuelve la mejor coincidencia: primero
// las que traen nota, y entre esas la de id menor. Sin este orden la fila
// elegida dependería del orden que devolviera MariaDB (la query no lleva ORDER
// BY) y de si respondió el camino rápido o el escaneo completo.
func pickOfficialRow(results []models.ExamOfficialResult, name, surname, fullDNI string) *models.ExamOfficialResult {
	var best *models.ExamOfficialResult
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

		// DNI: comparamos el DNI completo contra la máscara oficial, respetando
		// las posiciones que esa máscara revela (sin asumir una ventana fija).
		if !helpers.MatchMaskedDNI(res.DniMasked, fullDNI) {
			continue
		}

		if betterOfficialRow(best, &res) {
			best = &res
		}
	}

	return best
}

// betterOfficialRow decide si cand desplaza a la mejor fila encontrada hasta
// ahora: gana la que tiene nota; a igualdad, la de id menor.
func betterOfficialRow(best, cand *models.ExamOfficialResult) bool {
	if best == nil {
		return true
	}
	if (best.Score == nil) != (cand.Score == nil) {
		return cand.Score != nil
	}
	return cand.ID < best.ID
}

// ScoringConfigFromExam resolves an exam's scoring fields (with nil pointers)
// into a scoring.Config.
func ScoringConfigFromExam(e *models.Exam) scoring.Config {
	return scoring.ConfigFromExamFields(
		e.ScoringMode, e.SubtractsPoints,
		e.PenaltyValue, e.MaxScore, e.PointsPerCorrect, e.PointsPerWrong, e.WrongBlockSize,
	)
}

// CalculateScoreBreakdown is the legacy 5-arg entry point kept so existing
// callers compile unchanged. It computes a LEGACY-mode breakdown.
func CalculateScoreBreakdown(questions []models.Question, answers map[uint]string, subtracts bool, penalty float64, maxScore float64) (*ScoreBreakdown, error) {
	return CalculateScoreBreakdownCfg(questions, answers, scoring.Config{
		Mode: "legacy", Subtracts: subtracts, Penalty: penalty, MaxScore: maxScore,
	})
}

// CalculateScoreBreakdownCfg counts answers once and computes the score per the
// resolved scoring config.
//
// The LEGACY branch stays UNROUNDED here to preserve exact create-time parity
// (RecalculateScores rounds later). The ABSOLUTE branch uses scoring.ComputeScore
// (rounded, floored at 0).
func CalculateScoreBreakdownCfg(questions []models.Question, answers map[uint]string, cfg scoring.Config) (*ScoreBreakdown, error) {
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

	var score float64
	switch cfg.Mode {
	case "absolute":
		score = scoring.ComputeScore(correct, incorrect, total, cfg)
	default: // legacy: UNROUNDED to preserve exact create-time parity.
		netCorrect := float64(correct)
		switch {
		case cfg.WrongBlockSize > 0:
			netCorrect -= math.Floor(float64(incorrect) / cfg.WrongBlockSize)
		case cfg.Subtracts && cfg.Penalty > 0:
			netCorrect -= float64(incorrect) * cfg.Penalty
		}
		score = netCorrect / float64(total) * cfg.MaxScore
	}

	notAnswered := total - correct - incorrect

	return &ScoreBreakdown{
		Score:            score,
		CorrectAnswers:   correct,
		IncorrectAnswers: incorrect,
		NotAnswered:      notAnswered,
		TotalQuestions:   total,
	}, nil
}

// CalculateGroupedBreakdown scores a submission per question group (absolute model
// per group, total = sum) via the single-source scoring.ComputeGrouped. The
// returned ScoreBreakdown carries the per-group outcomes so the payload layer can
// surface them and decide eliminatory pass/fail without reloading anything.
func CalculateGroupedBreakdown(groups []models.QuestionGroup, questions []models.Question, answers map[uint]string, wrongBlockSize float64) (*ScoreBreakdown, error) {
	return breakdownFromGrouped(scoring.ComputeGrouped(groups, questions, answers, wrongBlockSize))
}

// CalculateGroupedXuntaBreakdown is the Modo Xunta counterpart: same per-group
// packing, but the per-group grade comes from the piecewise-linear Xunta curve.
func CalculateGroupedXuntaBreakdown(groups []models.QuestionGroup, questions []models.Question, answers map[uint]string) (*ScoreBreakdown, error) {
	return breakdownFromGrouped(scoring.ComputeGroupedXunta(groups, questions, answers))
}

// breakdownFromGrouped packs a GroupedResult into a ScoreBreakdown shared by both
// grouped scorers so the payload layer surfaces per-group outcomes uniformly.
func breakdownFromGrouped(gr scoring.GroupedResult) (*ScoreBreakdown, error) {
	correct, incorrect, total := 0, 0, 0
	for _, g := range gr.Groups {
		correct += g.Correct
		incorrect += g.Incorrect
		total += g.Total
	}
	if total == 0 {
		return nil, errors.New("el examen no tiene preguntas activas no anuladas configuradas")
	}
	return &ScoreBreakdown{
		Score:            gr.Total,
		CorrectAnswers:   correct,
		IncorrectAnswers: incorrect,
		NotAnswered:      total - correct - incorrect,
		TotalQuestions:   total,
		Groups:           gr.Groups,
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

// findOfficialForSubmission busca la fila oficial del alumno. Primero por link
// directo (user_id) y, si no lo hay, repitiendo el matching por DNI/nombre del
// envío: quien entregó antes de que se importaran los resultados oficiales no
// tiene fila linkada y si no, seguiría viendo su nota estimada en vez de la
// oficial.
func findOfficialForSubmission(db *gorm.DB, exam *models.Exam, submission *models.UserExamSubmission) *models.ExamOfficialResult {
	if submission.UserID != 0 {
		var official models.ExamOfficialResult
		if err := db.Where("exam_id = ? AND user_id = ?", exam.ID, submission.UserID).
			Order("score IS NULL, id").
			First(&official).Error; err == nil {
			return &official
		}
	}
	if submission.User.DNI == "" {
		return nil
	}
	official, err := FindOfficialResult(db, OfficialResultMatchRequest{
		ExamID:  exam.ID,
		Name:    submission.User.Name,
		Surname: submission.User.Surname,
		DNI:     submission.User.DNI,
	})
	if err != nil {
		return nil
	}
	if official == nil {
		return nil
	}
	// Solo se acepta si nadie más la tiene asignada: el matching difuso no debe
	// robarle la nota a la persona a la que ya está linkada.
	if official.UserID != nil {
		if *official.UserID != submission.UserID {
			return nil
		}
		return official
	}
	// Se persiste el link para que el siguiente acceso salga por user_id: el
	// matching por DNI puede acabar escaneando el examen entero (>12k filas en
	// los más grandes) y esta función está en rutas públicas calientes. El
	// WHERE ... IS NULL evita pisar un link creado en paralelo.
	if submission.UserID != 0 {
		if err := db.Model(&models.ExamOfficialResult{}).
			Where("id = ? AND user_id IS NULL", official.ID).
			Update("user_id", submission.UserID).Error; err != nil {
			log.Printf("failed to link official result %d to user %d: %v", official.ID, submission.UserID, err)
		}
	}
	return official
}

// applyOfficialScoreOverride overrides submission score/merits in memory when
// a linked official result with a score exists. Devuelve true si la fila oficial
// encontrada trae nota. El orden es explícito (primero las filas con nota, luego
// por id) porque un alumno puede estar linkado a más de una fila.
func applyOfficialScoreOverride(db *gorm.DB, exam *models.Exam, submission *models.UserExamSubmission) bool {
	official := findOfficialForSubmission(db, exam, submission)
	if official == nil {
		return false
	}
	if official.Score != nil {
		v := *official.Score
		submission.Score = &v
	}
	if official.Merits != nil {
		v := *official.Merits
		submission.Merits = &v
	}
	return official.Score != nil
}

func (s *Service) buildSubmissionPayload(tx *gorm.DB, exam *models.Exam, submission *models.UserExamSubmission, message string, breakdown *ScoreBreakdown, examRepo repository.ExamRepository, submissionRepo repository.SubmissionRepository) (*SubmissionPayload, error) {
	hasOfficialScore := applyOfficialScoreOverride(tx, exam, submission)

	// Entrega "solo oficial": la nota viene del resultado oficial y no hay
	// respuestas que corregir. Calcular el breakdown daría 0 aciertos y N en
	// blanco (y tarjetas de grupo a cero), contradiciendo la nota que se muestra,
	// así que se omiten contadores, grupos y revisión de respuestas.
	isOfficialOnly := hasOfficialScore &&
		(submission.AnswersData == nil || len(*submission.AnswersData) == 0)

	// In absolute mode the effective maximum is points_per_correct * active
	// questions, not the stored legacy MaxScore (default 100). This keeps the
	// student's "nota sobre X" and secondary bases correct.
	// Detect grouped scoring. On the submit path the breakdown carries the group
	// outcomes; on the check path breakdown is nil, so recompute from the stored
	// answers (fetchScoreBreakdownFromDB loads the groups itself).
	// Sin respuestas no hay outcomes por grupo que mostrar.
	var groupOutcomes []scoring.GroupOutcome
	if !isOfficialOnly {
		if breakdown != nil && len(breakdown.Groups) > 0 {
			groupOutcomes = breakdown.Groups
		} else if len(exam.Groups) > 0 {
			bd, err := fetchScoreBreakdownFromDB(tx, exam, submission.AnswersData)
			if err != nil {
				return nil, err
			}
			if bd != nil {
				groupOutcomes = bd.Groups
			}
		}
	}
	isGrouped := len(groupOutcomes) > 0

	// In absolute/grouped mode the effective maximum is not the stored legacy
	// MaxScore (default 100): grouped = sum of group maxima, absolute =
	// points_per_correct * active questions.
	effectiveMax := exam.MaxScore
	if isOfficialOnly && len(exam.Groups) > 0 {
		// En agrupado la base sigue siendo la suma de valoraciones de grupo,
		// aunque no haya outcomes que calcular.
		var sum float64
		for _, g := range exam.Groups {
			sum += g.MaxScore
		}
		effectiveMax = &sum
	} else if isGrouped {
		var sum float64
		if exam.ScoringMode == "xunta" {
			// Xunta group outcomes carry the question count in MaxScore (for the
			// "netas / 80" card), so the total's "sobre 100" base comes from the
			// group valoraciones instead.
			for _, g := range exam.Groups {
				sum += g.MaxScore
			}
		} else {
			for _, g := range groupOutcomes {
				sum += g.MaxScore
			}
		}
		effectiveMax = &sum
	} else if exam.ScoringMode == "absolute" && exam.PointsPerCorrect != nil {
		n, err := examRepo.CountActiveQuestions(context.Background(), tx, exam.ID)
		if err != nil {
			return nil, err
		}
		m := *exam.PointsPerCorrect * float64(n)
		effectiveMax = &m
	}

	payload := &SubmissionPayload{
		Message:            message,
		Merits:             submission.Merits,
		MaxScore:           effectiveMax,
		SecondaryMaxScores: exam.SecondaryMaxScores,
		Groups:             groupOutcomes,
		IsOfficialOnly:     isOfficialOnly,
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

	if exam.ShowScoreFull && !isOfficialOnly {
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

	if exam.ValidatedTribunal && !isOfficialOnly {
		review, err := buildAnswersReview(tx, exam, submission)
		if err != nil {
			return nil, err
		}
		payload.AnswersReview = review
	}

	// Passing criteria logic. Grouped exams pass iff every eliminatory group met
	// its minimum (the global PassingThreshold does not apply to grouped exams).
	threshold := exam.PassingThreshold
	var isPassed *bool
	if isGrouped {
		gr := scoring.GroupedResult{Groups: groupOutcomes}
		p := gr.AllEliminatoryPassed()
		isPassed = &p
	} else {
		isPassed = evaluatePassStatus(submission.Score, threshold)
	}
	payload.IsPassed = isPassed
	payload.CanEditMerits = isPassed != nil && *isPassed
	payload.AllowMeritsEdit = exam.AllowMeritsEdit
	mm := exam.MaxMerits
	payload.MaxMerits = &mm

	if threshold != nil {
		var passedCount int64
		tx.Raw(`
			SELECT COUNT(DISTINCT s.id)
			FROM user_exam_submission s
			LEFT JOIN exam_official_result o
				ON o.exam_id = s.exam_id AND o.user_id = s.user_id AND o.score IS NOT NULL
			WHERE s.exam_id = ? AND COALESCE(o.score, s.score) >= ?`,
			exam.ID, *threshold).Scan(&passedCount)
		pc := int(passedCount)
		payload.PassedCount = &pc
	}

	if payload.CanEditMerits && submission.Merits != nil && threshold != nil {
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

	// La nota que cuenta es la oficial cuando existe, igual que en el percentil
	// (RecalculatePercentiles) y en el ranking de méritos. submission.Score ya
	// viene con el override aplicado.
	var better int64
	if err := tx.Raw(`
		SELECT COUNT(DISTINCT s.id)
		FROM user_exam_submission s
		LEFT JOIN exam_official_result o
			ON o.exam_id = s.exam_id AND o.user_id = s.user_id AND o.score IS NOT NULL
		WHERE s.exam_id = ? AND s.id != ? AND COALESCE(o.score, s.score) > ?`,
		submission.ExamID, submission.ID, *submission.Score).Scan(&better).Error; err != nil {
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

	groups := exam.Groups
	if len(groups) == 0 {
		if err := tx.Where("exam_id = ?", exam.ID).Find(&groups).Error; err != nil {
			return nil, err
		}
	}
	if len(groups) > 0 {
		if exam.ScoringMode == "xunta" {
			return CalculateGroupedXuntaBreakdown(groups, questions, answerMap)
		}
		return CalculateGroupedBreakdown(groups, questions, answerMap, ScoringConfigFromExam(exam).WrongBlockSize)
	}

	breakdown, err := CalculateScoreBreakdownCfg(questions, answerMap, ScoringConfigFromExam(exam))
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
			QuestionID:       question.ID,
			QuestionLabel:    effectiveQuestionLabel(question),
			SelectedOption:   selectedPtr,
			CorrectOption:    correctPtr,
			IsCorrect:        has && correctPtr != nil && selectedPtr != nil && strings.EqualFold(*selectedPtr, *correctPtr),
			HasFeedbackVideo: question.FeedbackVideoKey != nil,
		})
	}

	return review, nil
}
