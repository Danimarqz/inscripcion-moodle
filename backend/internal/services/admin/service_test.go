package admin

import (
	"context"
	"testing"

	"github.com/inscripcion-moodle/go-backend/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gorm.io/gorm"
)

// MockExamRepository is a mock implementation of repository.ExamRepository
type MockExamRepository struct {
	mock.Mock
}

func (m *MockExamRepository) FindExamByID(ctx context.Context, db *gorm.DB, examID uint) (*models.Exam, error) {
	args := m.Called(ctx, db, examID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Exam), args.Error(1)
}

func (m *MockExamRepository) FindExamByIDLite(ctx context.Context, db *gorm.DB, examID uint) (*models.Exam, error) {
	args := m.Called(ctx, db, examID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Exam), args.Error(1)
}

func (m *MockExamRepository) FindExamBySlug(ctx context.Context, db *gorm.DB, slug string) (*models.Exam, error) {
	args := m.Called(ctx, db, slug)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Exam), args.Error(1)
}

func (m *MockExamRepository) FindQuestionsByExamID(ctx context.Context, db *gorm.DB, examID uint) ([]models.Question, error) {
	args := m.Called(ctx, db, examID)
	return args.Get(0).([]models.Question), args.Error(1)
}

func (m *MockExamRepository) ListExams(ctx context.Context, db *gorm.DB) ([]models.Exam, error) {
	args := m.Called(ctx, db)
	return args.Get(0).([]models.Exam), args.Error(1)
}

func (m *MockExamRepository) CreateExam(ctx context.Context, db *gorm.DB, exam *models.Exam) error {
	args := m.Called(ctx, db, exam)
	return args.Error(0)
}

func (m *MockExamRepository) UpdateExam(ctx context.Context, db *gorm.DB, exam *models.Exam) error {
	args := m.Called(ctx, db, exam)
	return args.Error(0)
}

func (m *MockExamRepository) DeleteExam(ctx context.Context, db *gorm.DB, examID uint) error {
	args := m.Called(ctx, db, examID)
	return args.Error(0)
}

func (m *MockExamRepository) CountByName(ctx context.Context, db *gorm.DB, name string) (int64, error) {
	args := m.Called(ctx, db, name)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockExamRepository) RecalculateScores(ctx context.Context, db *gorm.DB, examID uint) error {
	args := m.Called(ctx, db, examID)
	return args.Error(0)
}

func (m *MockExamRepository) RecalculateScoresForSubmission(ctx context.Context, db *gorm.DB, examID uint, submissionID uint) error {
	args := m.Called(ctx, db, examID, submissionID)
	return args.Error(0)
}

func (m *MockExamRepository) RecalculatePercentiles(ctx context.Context, db *gorm.DB, examID uint) error {
	args := m.Called(ctx, db, examID)
	return args.Error(0)
}

func (m *MockExamRepository) SetPercentileGroup(ctx context.Context, db *gorm.DB, examID uint, memberIDs []uint) ([]uint, error) {
	args := m.Called(ctx, db, examID, memberIDs)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]uint), args.Error(1)
}

func (m *MockExamRepository) GetTop10AverageScore(ctx context.Context, db *gorm.DB, examID uint) (*float64, error) {
	args := m.Called(ctx, db, examID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*float64), args.Error(1)
}

func (m *MockExamRepository) CountActiveQuestions(ctx context.Context, db *gorm.DB, examID uint) (int, error) {
	args := m.Called(ctx, db, examID)
	return args.Int(0), args.Error(1)
}

func (m *MockExamRepository) FindAssociatedExamIDs(ctx context.Context, db *gorm.DB, examID uint, percentileGroup uint) ([]uint, error) {
	args := m.Called(ctx, db, examID, percentileGroup)
	return args.Get(0).([]uint), args.Error(1)
}

func (m *MockExamRepository) FindGroupsByExamID(ctx context.Context, db *gorm.DB, examID uint) ([]models.QuestionGroup, error) {
	args := m.Called(ctx, db, examID)
	return args.Get(0).([]models.QuestionGroup), args.Error(1)
}

type MockQuestionRepository struct {
	mock.Mock
}

func (m *MockQuestionRepository) FindByID(ctx context.Context, db *gorm.DB, id uint) (*models.Question, error) {
	args := m.Called(ctx, db, id)
	if q, ok := args.Get(0).(*models.Question); ok {
		return q, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockQuestionRepository) DeleteByExamID(ctx context.Context, db *gorm.DB, examID uint) error {
	args := m.Called(ctx, db, examID)
	return args.Error(0)
}

func (m *MockQuestionRepository) DeleteQuestions(ctx context.Context, db *gorm.DB, questionIDs []uint) error {
	args := m.Called(ctx, db, questionIDs)
	return args.Error(0)
}

func TestService_ListExams(t *testing.T) {
	mockRepo := new(MockExamRepository)
	service := New(nil, mockRepo, nil, nil, nil) // passing nil for questionRepo

	expectedExams := []models.Exam{{Name: "Test Exam"}}

	// Expect ListExams to be called
	mockRepo.On("ListExams", mock.Anything, mock.Anything).Return(expectedExams, nil)

	exams, err := service.ListExams()
	assert.NoError(t, err)
	assert.Equal(t, expectedExams, exams)
	mockRepo.AssertExpectations(t)
}

func TestService_GetExam(t *testing.T) {
	mockRepo := new(MockExamRepository)
	service := New(nil, mockRepo, nil, nil, nil)

	expectedExam := &models.Exam{Name: "Test Exam"}

	mockRepo.On("FindExamByIDLite", mock.Anything, mock.Anything, uint(1)).Return(expectedExam, nil)
	mockRepo.On("FindGroupsByExamID", mock.Anything, mock.Anything, uint(1)).Return([]models.QuestionGroup{}, nil)

	exam, err := service.GetExam(1)
	assert.NoError(t, err)
	assert.Equal(t, expectedExam.Name, exam.Name)
	mockRepo.AssertExpectations(t)
}

func TestService_GetExam_NotFound(t *testing.T) {
	mockRepo := new(MockExamRepository)
	service := New(nil, mockRepo, nil, nil, nil)

	mockRepo.On("FindExamByIDLite", mock.Anything, mock.Anything, uint(99)).Return(nil, gorm.ErrRecordNotFound)

	exam, err := service.GetExam(99)
	assert.Error(t, err)
	assert.Nil(t, exam)
	assert.Contains(t, err.Error(), "no existe")
	mockRepo.AssertExpectations(t)
}
