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

	// Request updates only Q101 and Q103. Q102 should be deleted.
	reqQ1 := QuestionInput{ID: new(uint(101)), Name: new(1), CorrectOption: "A"}
	reqQ3 := QuestionInput{ID: new(uint(103)), Name: new(3), CorrectOption: "B"}
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

func TestService_UpdateExam_Numbering_SEQ_Renumbering(t *testing.T) {
	mockExamRepo := new(MockExamRepository)
	mockQuestionRepo := new(MockQuestionRepository)
	service := New(nil, mockExamRepo, nil, nil, mockQuestionRepo)

	examID := uint(1)
	// User has questions 1, 3 (2 was deleted previously).
	existingQuestions := []models.Question{
		{ID: 201, ExamID: examID, Name: 1, IsActive: true},
		{ID: 203, ExamID: examID, Name: 3, IsActive: true},
	}
	exam := &models.Exam{ID: examID, Questions: existingQuestions}

	// Update sends same questions. System should renumber 1, 3 -> 1, 2.
	req := EditExamRequest{
		Questions: []QuestionInput{
			{ID: new(uint(201)), Name: new(1), CorrectOption: "A"},
			{ID: new(uint(203)), Name: new(3), CorrectOption: "B"},
		},
	}

	mockExamRepo.On("FindExamByID", mock.Anything, mock.Anything, examID).Return(exam, nil)
	mockExamRepo.On("UpdateExam", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	mockExamRepo.On("RecalculateScores", mock.Anything, mock.Anything, examID).Return(nil)

	updatedExam, err := service.UpdateExam(examID, req)
	assert.NoError(t, err)

	// Verify numbering re-sequenced
	q1 := findQuestionByID(updatedExam.Questions, 201)
	q3 := findQuestionByID(updatedExam.Questions, 203)

	assert.Equal(t, 1, q1.Name)
	assert.Equal(t, 2, q3.Name)
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

	// Add new question. Should get 3.
	req := EditExamRequest{
		Questions: []QuestionInput{
			{ID: new(uint(301)), Name: new(1), CorrectOption: "A"},
			{ID: new(uint(302)), Name: new(2), CorrectOption: "B"},
			{Name: nil, CorrectOption: "C"}, // New
		},
	}

	mockExamRepo.On("FindExamByID", mock.Anything, mock.Anything, examID).Return(exam, nil)
	mockExamRepo.On("UpdateExam", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	mockExamRepo.On("RecalculateScores", mock.Anything, mock.Anything, examID).Return(nil)

	updatedExam, err := service.UpdateExam(examID, req)
	assert.NoError(t, err)

	assert.Len(t, updatedExam.Questions, 3)
	// Find the new one (ID=0)
	var newQ models.Question
	for _, q := range updatedExam.Questions {
		if q.ID == 0 {
			newQ = q
			break
		}
	}
	assert.Equal(t, 3, newQ.Name)
}

func TestService_UpdateExam_ActiveReserveSeparation(t *testing.T) {
	mockExamRepo := new(MockExamRepository)
	mockQuestionRepo := new(MockQuestionRepository)
	service := New(nil, mockExamRepo, nil, nil, mockQuestionRepo)

	examID := uint(1)
	// Existing: Active 1, Reserve 1 (Old state)
	existingQuestions := []models.Question{
		{ID: 401, ExamID: examID, Name: 1, IsActive: true},
		{ID: 402, ExamID: examID, Name: 1, IsActive: false},
	}
	exam := &models.Exam{ID: examID, Questions: existingQuestions}

	// Add new Active and new Reserve.
	// Input order matters!
	// 1. ID 401 (Active) -> Name 1
	// 2. New Active       -> Name 2
	// 3. ID 402 (Reserve) -> Name 3
	// 4. New Reserve      -> Name 4
	req := EditExamRequest{
		Questions: []QuestionInput{
			{ID: new(uint(401)), Name: new(1), IsActive: new(true), CorrectOption: "A"},  // Active
			{IsActive: new(true), CorrectOption: "C"},                                    // New Active
			{ID: new(uint(402)), Name: new(1), IsActive: new(false), CorrectOption: "B"}, // Reserve
			{IsActive: new(false), CorrectOption: "D"},                                   // New Reserve
		},
	}

	mockExamRepo.On("FindExamByID", mock.Anything, mock.Anything, examID).Return(exam, nil)
	mockExamRepo.On("UpdateExam", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	mockExamRepo.On("RecalculateScores", mock.Anything, mock.Anything, examID).Return(nil)

	updatedExam, err := service.UpdateExam(examID, req)
	assert.NoError(t, err)

	// Verify names
	q401 := findQuestionByID(updatedExam.Questions, 401)
	q402 := findQuestionByID(updatedExam.Questions, 402)

	var newActive, newReserve models.Question
	for _, q := range updatedExam.Questions {
		if q.ID == 0 {
			if q.IsActive {
				newActive = q
			} else {
				newReserve = q
			}
		}
	}

	// Expect strict sequential ordering based on input list
	assert.Equal(t, 1, q401.Name, "First Active question should be 1")
	assert.Equal(t, 2, newActive.Name, "Second Active (New) should be 2")
	assert.Equal(t, 3, q402.Name, "First Reserve (Old) should be 3")
	assert.Equal(t, 4, newReserve.Name, "Second Reserve (New) should be 4")
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
