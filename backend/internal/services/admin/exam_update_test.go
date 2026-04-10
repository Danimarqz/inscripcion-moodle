package admin

import (
	"testing"

	"github.com/inscripcion-moodle/go-backend/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestService_UpdateExam_DeleteQuestions(t *testing.T) {
	mockExamRepo := new(MockExamRepository)
	mockQuestionRepo := new(MockQuestionRepository)
	service := New(nil, mockExamRepo, nil, nil, mockQuestionRepo)

	examID := uint(1)
	existingQuestions := []models.Question{
		{ID: 101, ExamID: examID, Name: 1, IsActive: true},
		{ID: 102, ExamID: examID, Name: 2, IsActive: true},
		{ID: 103, ExamID: examID, Name: 3, IsActive: true},
	}
	exam := &models.Exam{
		ID:        examID,
		Name:      "Test Exam",
		Questions: existingQuestions,
	}

	// Request updates only Q101 and Q103 (renumbered to 1, 2). Q102 should be deleted.
	reqQ1 := QuestionInput{ID: new(uint(101)), Name: new(1), CorrectOption: "A"}
	reqQ3 := QuestionInput{ID: new(uint(103)), Name: new(2), CorrectOption: "B"}
	req := EditExamRequest{
		Questions: []QuestionInput{reqQ1, reqQ3},
	}

	mockExamRepo.On("FindExamByID", mock.Anything, mock.Anything, examID).Return(exam, nil)
	mockQuestionRepo.On("DeleteQuestions", mock.Anything, mock.Anything, []uint{102}).Return(nil)
	mockExamRepo.On("UpdateExam", mock.Anything, mock.Anything, mock.AnythingOfType("*models.Exam")).Return(nil)
	mockExamRepo.On("RecalculateScores", mock.Anything, mock.Anything, examID).Return(nil)

	updatedExam, err := service.UpdateExam(examID, req)

	assert.NoError(t, err)
	assert.NotNil(t, updatedExam)
	assert.Len(t, updatedExam.Questions, 2)
	assert.Equal(t, uint(101), updatedExam.Questions[0].ID)
	assert.Equal(t, uint(103), updatedExam.Questions[1].ID)

	mockExamRepo.AssertExpectations(t)
	mockQuestionRepo.AssertExpectations(t)
}

func TestService_UpdateExam_Numbering_PreservesManualNames(t *testing.T) {
	mockExamRepo := new(MockExamRepository)
	mockQuestionRepo := new(MockQuestionRepository)
	service := New(nil, mockExamRepo, nil, nil, mockQuestionRepo)

	examID := uint(1)
	existingQuestions := []models.Question{
		{ID: 201, ExamID: examID, Name: 1, IsActive: true},
		{ID: 202, ExamID: examID, Name: 2, IsActive: true},
		{ID: 203, ExamID: examID, Name: 3, IsActive: true},
	}
	exam := &models.Exam{ID: examID, Questions: existingQuestions}

	// Delete the middle question and resubmit the remaining two as 1, 2.
	// The surviving name=3 must shift down to 2 (renumber performed by the
	// client, not the server) so that the set stays contiguous 1..N.
	req := EditExamRequest{
		Questions: []QuestionInput{
			{ID: new(uint(201)), Name: new(1), CorrectOption: "A"},
			{ID: new(uint(203)), Name: new(2), CorrectOption: "B"},
		},
	}

	mockExamRepo.On("FindExamByID", mock.Anything, mock.Anything, examID).Return(exam, nil)
	mockQuestionRepo.On("DeleteQuestions", mock.Anything, mock.Anything, []uint{202}).Return(nil)
	mockExamRepo.On("UpdateExam", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	mockExamRepo.On("RecalculateScores", mock.Anything, mock.Anything, examID).Return(nil)

	updatedExam, err := service.UpdateExam(examID, req)
	assert.NoError(t, err)

	q201 := findQuestionByID(updatedExam.Questions, 201)
	q203 := findQuestionByID(updatedExam.Questions, 203)
	assert.Equal(t, 1, q201.Name)
	assert.Equal(t, 2, q203.Name)
}

func TestService_UpdateExam_Numbering_RejectsGap(t *testing.T) {
	mockExamRepo := new(MockExamRepository)
	mockQuestionRepo := new(MockQuestionRepository)
	service := New(nil, mockExamRepo, nil, nil, mockQuestionRepo)

	examID := uint(1)
	existingQuestions := []models.Question{
		{ID: 210, ExamID: examID, Name: 1, IsActive: true},
		{ID: 211, ExamID: examID, Name: 2, IsActive: true},
		{ID: 212, ExamID: examID, Name: 3, IsActive: true},
	}
	exam := &models.Exam{ID: examID, Questions: existingQuestions}

	// Delete 211 and leave {1, 3} — a gap. Must be rejected.
	req := EditExamRequest{
		Questions: []QuestionInput{
			{ID: new(uint(210)), Name: new(1), CorrectOption: "A"},
			{ID: new(uint(212)), Name: new(3), CorrectOption: "B"},
		},
	}

	mockExamRepo.On("FindExamByID", mock.Anything, mock.Anything, examID).Return(exam, nil)

	_, err := service.UpdateExam(examID, req)
	assert.ErrorIs(t, err, ErrInvalidQuestionNumbers)
}

func TestService_UpdateExam_Numbering_NewQuestion(t *testing.T) {
	mockExamRepo := new(MockExamRepository)
	mockQuestionRepo := new(MockQuestionRepository)
	service := New(nil, mockExamRepo, nil, nil, mockQuestionRepo)

	examID := uint(1)
	existingQuestions := []models.Question{
		{ID: 301, ExamID: examID, Name: 1, IsActive: true},
		{ID: 302, ExamID: examID, Name: 2, IsActive: true},
	}
	exam := &models.Exam{ID: examID, Questions: existingQuestions}

	// Add a new question with an explicit name of 3. The service must keep
	// that exact value instead of auto-assigning.
	req := EditExamRequest{
		Questions: []QuestionInput{
			{ID: new(uint(301)), Name: new(1), CorrectOption: "A"},
			{ID: new(uint(302)), Name: new(2), CorrectOption: "B"},
			{Name: new(3), CorrectOption: "C"},
		},
	}

	mockExamRepo.On("FindExamByID", mock.Anything, mock.Anything, examID).Return(exam, nil)
	mockExamRepo.On("UpdateExam", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	mockExamRepo.On("RecalculateScores", mock.Anything, mock.Anything, examID).Return(nil)

	updatedExam, err := service.UpdateExam(examID, req)
	assert.NoError(t, err)
	assert.Len(t, updatedExam.Questions, 3)

	var newQ models.Question
	for _, q := range updatedExam.Questions {
		if q.ID == 0 {
			newQ = q
			break
		}
	}
	assert.Equal(t, 3, newQ.Name)
}

func TestService_UpdateExam_InterleavedReserve(t *testing.T) {
	mockExamRepo := new(MockExamRepository)
	mockQuestionRepo := new(MockQuestionRepository)
	service := New(nil, mockExamRepo, nil, nil, mockQuestionRepo)

	examID := uint(1)
	existingQuestions := []models.Question{
		{ID: 401, ExamID: examID, Name: 1, IsActive: true},
		{ID: 402, ExamID: examID, Name: 2, IsActive: false},
		{ID: 403, ExamID: examID, Name: 3, IsActive: true},
	}
	exam := &models.Exam{ID: examID, Questions: existingQuestions}

	// Submit the reserve at name=2 interleaved between two actives at 1 and 3.
	// Input array order deliberately shuffled to prove it is ignored — the
	// stored Name comes from the explicit input.Name, not the slice index.
	req := EditExamRequest{
		Questions: []QuestionInput{
			{ID: new(uint(403)), Name: new(3), IsActive: new(true), CorrectOption: "C"},
			{ID: new(uint(401)), Name: new(1), IsActive: new(true), CorrectOption: "A"},
			{ID: new(uint(402)), Name: new(2), IsActive: new(false), CorrectOption: "B"},
		},
	}

	mockExamRepo.On("FindExamByID", mock.Anything, mock.Anything, examID).Return(exam, nil)
	mockExamRepo.On("UpdateExam", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	mockExamRepo.On("RecalculateScores", mock.Anything, mock.Anything, examID).Return(nil)

	updatedExam, err := service.UpdateExam(examID, req)
	assert.NoError(t, err)

	q401 := findQuestionByID(updatedExam.Questions, 401)
	q402 := findQuestionByID(updatedExam.Questions, 402)
	q403 := findQuestionByID(updatedExam.Questions, 403)

	assert.Equal(t, 1, q401.Name)
	assert.True(t, q401.IsActive)
	assert.Equal(t, 2, q402.Name)
	assert.False(t, q402.IsActive, "reserve sits interleaved at position 2")
	assert.Equal(t, 3, q403.Name)
	assert.True(t, q403.IsActive)
}

func TestService_UpdateExam_Numbering_RejectsMissingName(t *testing.T) {
	mockExamRepo := new(MockExamRepository)
	mockQuestionRepo := new(MockQuestionRepository)
	service := New(nil, mockExamRepo, nil, nil, mockQuestionRepo)

	examID := uint(1)
	exam := &models.Exam{
		ID: examID,
		Questions: []models.Question{
			{ID: 501, ExamID: examID, Name: 1, IsActive: true},
		},
	}

	req := EditExamRequest{
		Questions: []QuestionInput{
			{ID: new(uint(501)), Name: nil, CorrectOption: "A"},
		},
	}

	mockExamRepo.On("FindExamByID", mock.Anything, mock.Anything, examID).Return(exam, nil)

	_, err := service.UpdateExam(examID, req)
	assert.ErrorIs(t, err, ErrInvalidQuestionNumbers)
}

func TestService_UpdateExam_Numbering_RejectsDuplicate(t *testing.T) {
	mockExamRepo := new(MockExamRepository)
	mockQuestionRepo := new(MockQuestionRepository)
	service := New(nil, mockExamRepo, nil, nil, mockQuestionRepo)

	examID := uint(1)
	exam := &models.Exam{
		ID: examID,
		Questions: []models.Question{
			{ID: 601, ExamID: examID, Name: 1, IsActive: true},
			{ID: 602, ExamID: examID, Name: 2, IsActive: true},
		},
	}

	req := EditExamRequest{
		Questions: []QuestionInput{
			{ID: new(uint(601)), Name: new(1), CorrectOption: "A"},
			{ID: new(uint(602)), Name: new(1), CorrectOption: "B"},
		},
	}

	mockExamRepo.On("FindExamByID", mock.Anything, mock.Anything, examID).Return(exam, nil)

	_, err := service.UpdateExam(examID, req)
	assert.ErrorIs(t, err, ErrInvalidQuestionNumbers)
}

// Helper functions uintPtr, intPtr, boolPtr removed in favor of Go 1.26 new(value) syntax
// func uintPtr(v uint) *uint { return &v }
// func intPtr(v int) *int { return &v }
// func boolPtr(v bool) *bool { return &v }

func findQuestionByID(qs []models.Question, id uint) models.Question {
	for _, q := range qs {
		if q.ID == id {
			return q
		}
	}
	return models.Question{}
}
