package exam

import (
	"testing"

	"github.com/inscripcion-moodle/go-backend/internal/models"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// seedMeritsSubmission crea un alumno aprobado con méritos ya guardados.
func seedMeritsSubmission(t *testing.T, db *gorm.DB, allowEdit bool) *models.Exam {
	t.Helper()
	exam := seedOfficialOnlyExam(t, db)
	exam.PassingThreshold = new(50.0)
	exam.AllowMeritsEdit = allowEdit
	require.NoError(t, db.Save(exam).Error)

	user := &models.ExamUser{Email: "maria@example.com", DNI: "12345678Z", Name: "MARIA", Surname: "GARCIA"}
	require.NoError(t, db.Create(user).Error)
	require.NoError(t, db.Create(&models.UserExamSubmission{
		ExamID: exam.ID, UserID: user.ID, Score: new(80.0), Merits: new(5.0),
	}).Error)
	return exam
}

func meritsRequest(examID uint, merits float64) UpdateMeritsRequest {
	return UpdateMeritsRequest{DNI: "12345678Z", Email: "maria@example.com", ExamID: examID, Merits: &merits}
}

func TestUpdateMeritsRejectsSecondWriteByDefault(t *testing.T) {
	db := newOfficialOnlyDB(t)
	exam := seedMeritsSubmission(t, db, false)

	_, err := newOfficialOnlyService(db).UpdateMerits(meritsRequest(exam.ID, 7))
	require.ErrorIs(t, err, ErrMeritsAlreadySet)
}

func TestUpdateMeritsAllowsRewriteWhenExamAllowsIt(t *testing.T) {
	db := newOfficialOnlyDB(t)
	exam := seedMeritsSubmission(t, db, true)

	resp, err := newOfficialOnlyService(db).UpdateMerits(meritsRequest(exam.ID, 7))
	require.NoError(t, err)
	require.NotNil(t, resp.Merits)
	require.Equal(t, 7.0, *resp.Merits)

	var stored models.UserExamSubmission
	require.NoError(t, db.First(&stored).Error)
	require.Equal(t, 7.0, *stored.Merits)
}
