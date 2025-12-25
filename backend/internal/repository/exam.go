package repository

import (
	"context"

	"github.com/inscripcion-moodle/go-backend/internal/models"
	"gorm.io/gorm"
)

type ExamRepository interface {
	FindExamByID(ctx context.Context, db *gorm.DB, examID uint) (*models.Exam, error)
	FindQuestionsByExamID(ctx context.Context, db *gorm.DB, examID uint) ([]models.Question, error)
}

type examRepository struct{}

func NewExamRepository() ExamRepository {
	return &examRepository{}
}

func (r *examRepository) FindExamByID(ctx context.Context, db *gorm.DB, examID uint) (*models.Exam, error) {
	var exam models.Exam
	if err := db.WithContext(ctx).First(&exam, examID).Error; err != nil {
		return nil, err
	}
	return &exam, nil
}

func (r *examRepository) FindQuestionsByExamID(ctx context.Context, db *gorm.DB, examID uint) ([]models.Question, error) {
	var questions []models.Question
	if err := db.WithContext(ctx).Where("exam_id = ?", examID).Order("name asc, id asc").Find(&questions).Error; err != nil {
		return nil, err
	}
	return questions, nil
}
