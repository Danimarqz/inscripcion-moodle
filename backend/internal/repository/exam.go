package repository

import (
	"context"

	"github.com/inscripcion-moodle/go-backend/internal/models"
	"gorm.io/gorm"
)

type ExamRepository interface {
	FindExamByID(ctx context.Context, db *gorm.DB, examID uint) (*models.Exam, error)
	FindQuestionsByExamID(ctx context.Context, db *gorm.DB, examID uint) ([]models.Question, error)
	ListExams(ctx context.Context, db *gorm.DB) ([]models.Exam, error)
	CreateExam(ctx context.Context, db *gorm.DB, exam *models.Exam) error
	UpdateExam(ctx context.Context, db *gorm.DB, exam *models.Exam) error
	DeleteExam(ctx context.Context, db *gorm.DB, examID uint) error
	CountByName(ctx context.Context, db *gorm.DB, name string) (int64, error)
}

type examRepository struct{}

func NewExamRepository() ExamRepository {
	return &examRepository{}
}

func (r *examRepository) FindExamByID(ctx context.Context, db *gorm.DB, examID uint) (*models.Exam, error) {
	var exam models.Exam
	if err := db.WithContext(ctx).Preload("Questions").First(&exam, examID).Error; err != nil {
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

func (r *examRepository) ListExams(ctx context.Context, db *gorm.DB) ([]models.Exam, error) {
	var exams []models.Exam
	if err := db.WithContext(ctx).Preload("Questions").Find(&exams).Error; err != nil {
		return nil, err
	}
	return exams, nil
}

func (r *examRepository) CreateExam(ctx context.Context, db *gorm.DB, exam *models.Exam) error {
	return db.WithContext(ctx).Create(exam).Error
}

func (r *examRepository) UpdateExam(ctx context.Context, db *gorm.DB, exam *models.Exam) error {
	return db.WithContext(ctx).Session(&gorm.Session{FullSaveAssociations: true}).Save(exam).Error
}

func (r *examRepository) DeleteExam(ctx context.Context, db *gorm.DB, examID uint) error {
	// Transaction logic is usually handled by the service wrapping this or passing a tx *gorm.DB
	// But individual atomic deletes can be here. The complex cascade delete fits better in the service orchestrating repositories, 
	// OR we put the whole logic here if we treat "Exam" as an aggregate root.
	// For now, simple helpers.
	return db.WithContext(ctx).Delete(&models.Exam{}, examID).Error
}

func (r *examRepository) CountByName(ctx context.Context, db *gorm.DB, name string) (int64, error) {
	var count int64
	err := db.WithContext(ctx).Model(&models.Exam{}).Where("name = ?", name).Count(&count).Error
	return count, err
}
