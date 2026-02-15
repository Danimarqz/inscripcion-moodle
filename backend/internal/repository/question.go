package repository

import (
	"context"

	"github.com/inscripcion-moodle/go-backend/internal/models"
	"gorm.io/gorm"
)

type QuestionRepository interface {
	DeleteByExamID(ctx context.Context, db *gorm.DB, examID uint) error
	DeleteQuestions(ctx context.Context, db *gorm.DB, questionIDs []uint) error
}

type questionRepository struct{}

func NewQuestionRepository() QuestionRepository {
	return &questionRepository{}
}

func (r *questionRepository) DeleteByExamID(ctx context.Context, db *gorm.DB, examID uint) error {
	return db.WithContext(ctx).Where("exam_id = ?", examID).Delete(&models.Question{}).Error
}

func (r *questionRepository) DeleteQuestions(ctx context.Context, db *gorm.DB, questionIDs []uint) error {
	if len(questionIDs) == 0 {
		return nil
	}
	return db.WithContext(ctx).Delete(&models.Question{}, questionIDs).Error
}
