package exam

import (
	"testing"

	"github.com/inscripcion-moodle/go-backend/internal/models"
	"github.com/stretchr/testify/require"
)

// Reproduce el caso de producción: el alumno entregó antes de que se importaran
// los resultados oficiales, así que su fila oficial no tiene user_id y el
// override por link directo no la encuentra.
func TestOfficialOverrideMatchesUnlinkedRow(t *testing.T) {
	db := newOfficialOnlyDB(t)
	exam := seedOfficialOnlyExam(t, db)

	user := &models.ExamUser{Email: "maria@example.com", DNI: "12345678Z", Name: "MARIA", Surname: "GARCIA LOPEZ"}
	require.NoError(t, db.Create(user).Error)
	submission := &models.UserExamSubmission{ExamID: exam.ID, UserID: user.ID, Score: new(0.0)}
	require.NoError(t, db.Create(submission).Error)

	official := officialRow(exam.ID, new(76.67), nil) // user_id NULL
	require.NoError(t, db.Create(official).Error)

	submission.User = *user
	require.True(t, applyOfficialScoreOverride(db, exam, submission))
	require.Equal(t, 76.67, *submission.Score)
}

// Una fila oficial ya asignada a otra persona no se le aplica a un tercero.
func TestOfficialOverrideIgnoresRowLinkedToAnotherUser(t *testing.T) {
	db := newOfficialOnlyDB(t)
	exam := seedOfficialOnlyExam(t, db)

	other := &models.ExamUser{Email: "otra@example.com", DNI: "87654321X", Name: "OTRA", Surname: "PERSONA"}
	require.NoError(t, db.Create(other).Error)
	official := officialRow(exam.ID, new(76.67), nil)
	official.UserID = &other.ID
	require.NoError(t, db.Create(official).Error)

	user := &models.ExamUser{Email: "maria@example.com", DNI: "12345678Z", Name: "MARIA", Surname: "GARCIA LOPEZ"}
	require.NoError(t, db.Create(user).Error)
	submission := &models.UserExamSubmission{ExamID: exam.ID, UserID: user.ID, Score: new(0.0), User: *user}

	require.False(t, applyOfficialScoreOverride(db, exam, submission))
	require.Equal(t, 0.0, *submission.Score)
}

// El fallback por DNI escanea el examen entero en el peor caso, así que el link
// se persiste: la segunda consulta debe salir ya por user_id.
func TestOfficialOverridePersistsLink(t *testing.T) {
	db := newOfficialOnlyDB(t)
	exam := seedOfficialOnlyExam(t, db)

	user := &models.ExamUser{Email: "maria@example.com", DNI: "12345678Z", Name: "MARIA", Surname: "GARCIA LOPEZ"}
	require.NoError(t, db.Create(user).Error)
	require.NoError(t, db.Create(officialRow(exam.ID, new(76.67), nil)).Error)

	submission := &models.UserExamSubmission{ExamID: exam.ID, UserID: user.ID, Score: new(0.0), User: *user}
	require.True(t, applyOfficialScoreOverride(db, exam, submission))

	var official models.ExamOfficialResult
	require.NoError(t, db.First(&official).Error)
	require.NotNil(t, official.UserID)
	require.Equal(t, user.ID, *official.UserID)
}

// La posición se calcula sobre la nota oficial cuando existe, no sobre la que
// el alumno estimó: si no, quien entregó un 0 y sacó un 76 oficial aparecería
// por detrás de todos.
func TestPositionUsesOfficialScoreOfOthers(t *testing.T) {
	db := newOfficialOnlyDB(t)
	exam := seedOfficialOnlyExam(t, db)

	// Rival: entregó 10 estimado, pero su oficial es 90.
	rival := &models.ExamUser{Email: "rival@example.com", DNI: "87654321X", Name: "OTRA", Surname: "PERSONA"}
	require.NoError(t, db.Create(rival).Error)
	rivalSub := &models.UserExamSubmission{ExamID: exam.ID, UserID: rival.ID, Score: new(10.0)}
	require.NoError(t, db.Create(rivalSub).Error)
	rivalOfficial := officialRow(exam.ID, new(90.0), nil)
	rivalOfficial.UserID = &rival.ID
	require.NoError(t, db.Create(rivalOfficial).Error)

	user := &models.ExamUser{Email: "maria@example.com", DNI: "12345678Z", Name: "MARIA", Surname: "GARCIA LOPEZ"}
	require.NoError(t, db.Create(user).Error)
	sub := &models.UserExamSubmission{ExamID: exam.ID, UserID: user.ID, Score: new(76.67)}
	require.NoError(t, db.Create(sub).Error)

	pos, total, err := getSubmissionPositionData(db, sub)
	require.NoError(t, err)
	require.Equal(t, 2, *total)
	require.Equal(t, 2, *pos) // el rival cuenta con su 90 oficial, no con su 10
}
