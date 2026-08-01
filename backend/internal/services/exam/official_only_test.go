package exam

import (
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/inscripcion-moodle/go-backend/internal/models"
	"github.com/inscripcion-moodle/go-backend/internal/repository"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// newOfficialOnlyDB monta un esquema mínimo en sqlite. dni_search es una columna
// generada STORED de MariaDB que no existe en los modelos, así que se añade a
// mano con valor vacío: el camino rápido de FindOfficialResult siempre incluye
// la cadena vacía entre los candidatos, de modo que encuentra estas filas igual.
func newOfficialOnlyDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&models.Exam{},
		&models.ExamUser{},
		&models.UserExamSubmission{},
		&models.ExamOfficialResult{},
		&models.Question{},
		&models.QuestionGroup{},
	))
	require.NoError(t, db.Exec("ALTER TABLE exam_official_result ADD COLUMN dni_search TEXT NOT NULL DEFAULT ''").Error)
	return db
}

func newOfficialOnlyService(db *gorm.DB) *Service {
	return NewService(db, repository.NewExamRepository(), repository.NewSubmissionRepository(), "info@example.com")
}

func seedOfficialOnlyExam(t *testing.T, db *gorm.DB) *models.Exam {
	t.Helper()
	exam := &models.Exam{
		Name:           "Oficial",
		IsActive:       true,
		ShowScore:      true,
		ShowScoreFull:  true,
		ShowPercentile: true,
		MaxScore:       new(100.0),
		ExamWeight:     0.6,
		MaxMerits:      40,
	}
	require.NoError(t, db.Create(exam).Error)
	require.NoError(t, db.Create(&models.Question{
		ExamID: exam.ID, Name: 1, CorrectOption: "A", IsActive: true,
	}).Error)
	return exam
}

func officialOnlyRequest(examID uint) SubmitExamRequest {
	return SubmitExamRequest{
		Email:            "maria@example.com",
		DNI:              "12345678Z",
		Name:             "MARIA",
		Surname:          "GARCIA LOPEZ",
		ExamID:           examID,
		AcceptsMarketing: true,
		ResultType:       "General",
	}
}

func officialRow(examID uint, score, merits *float64) *models.ExamOfficialResult {
	return &models.ExamOfficialResult{
		ExamID:     examID,
		DniMasked:  "***4567**",
		Apellido1:  "GARCIA",
		Apellido2:  new("LOPEZ"),
		Nombre:     "MARIA",
		ResultType: "Discapacidad",
		Score:      score,
		Merits:     merits,
	}
}

func submit(t *testing.T, svc *Service, db *gorm.DB, req SubmitExamRequest) (*SubmissionPayload, error) {
	t.Helper()
	var payload *SubmissionPayload
	err := db.Transaction(func(tx *gorm.DB) error {
		p, err := svc.processSubmission(tx, req, svc.contactEmail, svc.repo, svc.submissionRepo)
		if err != nil {
			return err
		}
		payload = p
		return nil
	})
	return payload, err
}

func TestProcessSubmissionUsesOfficialScoreWithoutAnswers(t *testing.T) {
	db := newOfficialOnlyDB(t)
	exam := seedOfficialOnlyExam(t, db)
	require.NoError(t, db.Create(officialRow(exam.ID, new(78.5), nil)).Error)

	payload, err := submit(t, newOfficialOnlyService(db), db, officialOnlyRequest(exam.ID))
	require.NoError(t, err)
	require.True(t, payload.IsOfficialOnly)
	require.NotNil(t, payload.Score)
	require.Equal(t, 78.5, *payload.Score)
	// Sin respuestas no se envían contadores: mostrarían 0 aciertos y contradirían la nota.
	require.Nil(t, payload.CorrectAnswers)
	require.Nil(t, payload.NotAnswered)

	var stored models.UserExamSubmission
	require.NoError(t, db.First(&stored).Error)
	require.NotNil(t, stored.Score)
	require.Equal(t, 78.5, *stored.Score)
	require.NotNil(t, stored.AnswersData)
	require.Empty(t, *stored.AnswersData)
	// La convocatoria sale de la fila oficial, no de la que eligió el alumno.
	require.Equal(t, "Discapacidad", stored.SelectedResultType)

	// La fila oficial queda linkada al alumno (percentil y override la buscan por user_id).
	var official models.ExamOfficialResult
	require.NoError(t, db.First(&official).Error)
	require.NotNil(t, official.UserID)
	require.Equal(t, stored.UserID, *official.UserID)
}

func TestProcessSubmissionKeepsOfficialMeritsEditable(t *testing.T) {
	db := newOfficialOnlyDB(t)
	exam := seedOfficialOnlyExam(t, db)
	require.NoError(t, db.Create(officialRow(exam.ID, new(78.5), new(12.0))).Error)

	payload, err := submit(t, newOfficialOnlyService(db), db, officialOnlyRequest(exam.ID))
	require.NoError(t, err)
	require.NotNil(t, payload.Merits)
	require.Equal(t, 12.0, *payload.Merits)

	// Los méritos oficiales se aplican en memoria, no se persisten: si se
	// guardaran, UpdateMerits fallaría con ErrMeritsAlreadySet para siempre.
	var stored models.UserExamSubmission
	require.NoError(t, db.First(&stored).Error)
	require.Nil(t, stored.Merits)
}

func TestProcessSubmissionStoresUserMeritsWhenOfficialHasNone(t *testing.T) {
	db := newOfficialOnlyDB(t)
	exam := seedOfficialOnlyExam(t, db)
	require.NoError(t, db.Create(officialRow(exam.ID, new(78.5), nil)).Error)

	req := officialOnlyRequest(exam.ID)
	req.Merits = new(9.25)
	_, err := submit(t, newOfficialOnlyService(db), db, req)
	require.NoError(t, err)

	var stored models.UserExamSubmission
	require.NoError(t, db.First(&stored).Error)
	require.NotNil(t, stored.Merits)
	require.Equal(t, 9.25, *stored.Merits)
}

func TestProcessSubmissionScoresAnswersWhenOfficialHasNoScore(t *testing.T) {
	db := newOfficialOnlyDB(t)
	exam := seedOfficialOnlyExam(t, db)
	require.NoError(t, db.Create(officialRow(exam.ID, nil, nil)).Error)

	req := officialOnlyRequest(exam.ID)
	var question models.Question
	require.NoError(t, db.First(&question).Error)
	req.Answers = []AnswerSubmission{{QuestionID: question.ID, Answer: "A"}}

	payload, err := submit(t, newOfficialOnlyService(db), db, req)
	require.NoError(t, err)
	require.False(t, payload.IsOfficialOnly)

	var stored models.UserExamSubmission
	require.NoError(t, db.First(&stored).Error)
	require.NotNil(t, stored.AnswersData)
	require.Len(t, *stored.AnswersData, 1)
}

func TestProcessSubmissionPrefersOfficialRowWithScore(t *testing.T) {
	db := newOfficialOnlyDB(t)
	exam := seedOfficialOnlyExam(t, db)
	// La fila sin nota se inserta primero: sin criterio explícito ganaría ella.
	require.NoError(t, db.Create(officialRow(exam.ID, nil, nil)).Error)
	require.NoError(t, db.Create(officialRow(exam.ID, new(64.0), nil)).Error)

	payload, err := submit(t, newOfficialOnlyService(db), db, officialOnlyRequest(exam.ID))
	require.NoError(t, err)
	require.True(t, payload.IsOfficialOnly)
	require.NotNil(t, payload.Score)
	require.Equal(t, 64.0, *payload.Score)
}

func TestProcessSubmissionOfficialOnlyRequiresRaffleAcceptance(t *testing.T) {
	db := newOfficialOnlyDB(t)
	exam := seedOfficialOnlyExam(t, db)
	require.NoError(t, db.Model(exam).Update("raffle_enabled", true).Error)
	require.NoError(t, db.Create(officialRow(exam.ID, new(78.5), nil)).Error)

	_, err := submit(t, newOfficialOnlyService(db), db, officialOnlyRequest(exam.ID))
	require.ErrorIs(t, err, ErrRaffleNotAccepted)

	var count int64
	require.NoError(t, db.Model(&models.UserExamSubmission{}).Count(&count).Error)
	require.Zero(t, count)
}

func TestProcessSubmissionOfficialOnlyRejectsSecondSubmission(t *testing.T) {
	db := newOfficialOnlyDB(t)
	exam := seedOfficialOnlyExam(t, db)
	require.NoError(t, db.Create(officialRow(exam.ID, new(78.5), nil)).Error)
	svc := newOfficialOnlyService(db)

	_, err := submit(t, svc, db, officialOnlyRequest(exam.ID))
	require.NoError(t, err)

	_, err = submit(t, svc, db, officialOnlyRequest(exam.ID))
	require.ErrorContains(t, err, "ya has enviado este examen")

	var count int64
	require.NoError(t, db.Model(&models.UserExamSubmission{}).Count(&count).Error)
	require.Equal(t, int64(1), count)
}
