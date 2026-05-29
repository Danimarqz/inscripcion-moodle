package admin

import (
	"testing"

	"github.com/inscripcion-moodle/go-backend/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestService_UpdateExam_AbsoluteScoring_Persists(t *testing.T) {
	mockExamRepo := new(MockExamRepository)
	mockQuestionRepo := new(MockQuestionRepository)
	service := New(nil, mockExamRepo, nil, nil, mockQuestionRepo)

	examID := uint(1)
	exam := &models.Exam{
		ID:   examID,
		Name: "Cantabria",
		Questions: []models.Question{
			{ID: 1, ExamID: examID, Name: 1, IsActive: true, CorrectOption: "A"},
			{ID: 2, ExamID: examID, Name: 2, IsActive: true, CorrectOption: "B"},
		},
	}
	req := EditExamRequest{
		ScoringMode:      new("absolute"),
		PointsPerCorrect: new(0.40),
		PointsPerWrong:   new(0.10),
		Questions: []QuestionInput{
			{ID: new(uint(1)), Name: new(1), CorrectOption: "A"},
			{ID: new(uint(2)), Name: new(2), CorrectOption: "B"},
		},
	}

	mockExamRepo.On("FindExamByID", mock.Anything, mock.Anything, examID).Return(exam, nil)
	mockExamRepo.On("UpdateExam", mock.Anything, mock.Anything, mock.AnythingOfType("*models.Exam")).Return(nil)
	mockExamRepo.On("RecalculateScores", mock.Anything, mock.Anything, examID).Return(nil)

	updated, err := service.UpdateExam(examID, req)
	assert.NoError(t, err)
	assert.Equal(t, "absolute", updated.ScoringMode)
	if assert.NotNil(t, updated.PointsPerCorrect) {
		assert.Equal(t, 0.40, *updated.PointsPerCorrect)
	}
	if assert.NotNil(t, updated.PointsPerWrong) {
		assert.Equal(t, 0.10, *updated.PointsPerWrong)
	}
	mockExamRepo.AssertExpectations(t) // RecalculateScores must fire (scoring_mode changed)
}

func TestService_UpdateExam_AbsoluteScoring_PpwZeroAllowed(t *testing.T) {
	mockExamRepo := new(MockExamRepository)
	service := New(nil, mockExamRepo, nil, nil, new(MockQuestionRepository))

	examID := uint(1)
	exam := &models.Exam{ID: examID, Questions: []models.Question{
		{ID: 1, ExamID: examID, Name: 1, IsActive: true, CorrectOption: "A"},
	}}
	req := EditExamRequest{
		ScoringMode:      new("absolute"),
		PointsPerCorrect: new(0.40),
		PointsPerWrong:   new(0.0), // explicit zero deduction is valid
		Questions:        []QuestionInput{{ID: new(uint(1)), Name: new(1), CorrectOption: "A"}},
	}
	mockExamRepo.On("FindExamByID", mock.Anything, mock.Anything, examID).Return(exam, nil)
	mockExamRepo.On("UpdateExam", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	mockExamRepo.On("RecalculateScores", mock.Anything, mock.Anything, examID).Return(nil)

	_, err := service.UpdateExam(examID, req)
	assert.NoError(t, err)
}

func TestService_UpdateExam_AbsoluteScoring_Rejected(t *testing.T) {
	cases := []struct {
		name string
		req  EditExamRequest
		want error
	}{
		{"missing points", EditExamRequest{ScoringMode: new("absolute")}, ErrAbsoluteScoringConfig},
		{"negative ppc", EditExamRequest{ScoringMode: new("absolute"), PointsPerCorrect: new(-0.1), PointsPerWrong: new(0.1)}, ErrAbsoluteScoringConfig},
		{"negative ppw", EditExamRequest{ScoringMode: new("absolute"), PointsPerCorrect: new(0.4), PointsPerWrong: new(-0.1)}, ErrAbsoluteScoringConfig},
		{"invalid mode", EditExamRequest{ScoringMode: new("bogus")}, ErrInvalidScoringMode},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mockExamRepo := new(MockExamRepository)
			service := New(nil, mockExamRepo, nil, nil, new(MockQuestionRepository))
			examID := uint(1)
			exam := &models.Exam{ID: examID, Questions: []models.Question{
				{ID: 1, ExamID: examID, Name: 1, IsActive: true, CorrectOption: "A"},
			}}
			mockExamRepo.On("FindExamByID", mock.Anything, mock.Anything, examID).Return(exam, nil)

			_, err := service.UpdateExam(examID, tc.req)
			assert.ErrorIs(t, err, tc.want)
			// Validation must short-circuit before any write/recalc.
			mockExamRepo.AssertNotCalled(t, "UpdateExam", mock.Anything, mock.Anything, mock.Anything)
			mockExamRepo.AssertNotCalled(t, "RecalculateScores", mock.Anything, mock.Anything, examID)
		})
	}
}
