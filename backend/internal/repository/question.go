package repository

import (
	"context"

	"github.com/inscripcion-moodle/go-backend/internal/models"
	"gorm.io/gorm"
)

type QuestionRepository interface {
	DeleteByExamID(ctx context.Context, db *gorm.DB, examID uint) error
}

type questionRepository struct{}

func NewQuestionRepository() QuestionRepository {
	return &questionRepository{}
}

func (r *questionRepository) DeleteByExamID(ctx context.Context, db *gorm.DB, examID uint) error {
	return db.WithContext(ctx).Where("exam_id = ?", examID).Delete(&models.Question{}).Error
}
