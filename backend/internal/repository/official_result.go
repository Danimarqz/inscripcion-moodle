package repository

import (
	"context"
	"strings"

	"github.com/inscripcion-moodle/go-backend/internal/models"
	"gorm.io/gorm"
)

type OfficialResultRepository interface {
	Create(ctx context.Context, db *gorm.DB, result *models.ExamOfficialResult) error
	FindByExamAndDNI(ctx context.Context, db *gorm.DB, examID uint, dni string) (*models.ExamOfficialResult, error)
	List(ctx context.Context, db *gorm.DB, examID uint, offset, limit int, order string) ([]models.ExamOfficialResult, int64, error)
	DeleteByExamID(ctx context.Context, db *gorm.DB, examID uint) error
}

type officialResultRepository struct{}

func NewOfficialResultRepository() OfficialResultRepository {
	return &officialResultRepository{}
}

func (r *officialResultRepository) Create(ctx context.Context, db *gorm.DB, result *models.ExamOfficialResult) error {
	return db.WithContext(ctx).Create(result).Error
}

func (r *officialResultRepository) FindByExamAndDNI(ctx context.Context, db *gorm.DB, examID uint, dni string) (*models.ExamOfficialResult, error) {
	var result models.ExamOfficialResult
	err := db.WithContext(ctx).Where("exam_id = ? AND dni_masked = ?", examID, dni).First(&result).Error
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (r *officialResultRepository) List(ctx context.Context, db *gorm.DB, examID uint, offset, limit int, order string) ([]models.ExamOfficialResult, int64, error) {
	var results []models.ExamOfficialResult
	var total int64

	query := db.WithContext(ctx).Model(&models.ExamOfficialResult{}).Where("exam_id = ?", examID)
	
	if strings.Contains(strings.ToLower(order), "exam_user.") {
		query = query.Joins("LEFT JOIN exam_user ON exam_user.id = exam_official_result.user_id")
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if order != "" {
		query = query.Order(order)
	}
	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	if err := query.Preload("User").Find(&results).Error; err != nil {
		return nil, 0, err
	}
	return results, total, nil
}

func (r *officialResultRepository) DeleteByExamID(ctx context.Context, db *gorm.DB, examID uint) error {
	return db.WithContext(ctx).Where("exam_id = ?", examID).Delete(&models.ExamOfficialResult{}).Error
}
