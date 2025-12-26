package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/inscripcion-moodle/go-backend/internal/models"
	"gorm.io/gorm"
)

type SubmissionRepository interface {
	FindByID(ctx context.Context, db *gorm.DB, submissionID uint) (*models.UserExamSubmission, error)
	List(ctx context.Context, db *gorm.DB, examID uint, limit, offset int, search, order string, moodleSynced *bool) ([]models.UserExamSubmission, error)
	Count(ctx context.Context, db *gorm.DB, examID uint, moodleSynced *bool) (int64, error)
	Delete(ctx context.Context, db *gorm.DB, submissionID uint) error
	DeleteByExamID(ctx context.Context, db *gorm.DB, examID uint) error
	GetAverageScore(ctx context.Context, db *gorm.DB, examID uint) (*float64, error)
	Update(ctx context.Context, db *gorm.DB, submission *models.UserExamSubmission) error
	SaveAnswer(ctx context.Context, db *gorm.DB, answer *models.UserAnswer) error
	CreateAnswer(ctx context.Context, db *gorm.DB, answer *models.UserAnswer) error
}

type submissionRepository struct{}

func NewSubmissionRepository() SubmissionRepository {
	return &submissionRepository{}
}

func (r *submissionRepository) FindByID(ctx context.Context, db *gorm.DB, submissionID uint) (*models.UserExamSubmission, error) {
	var submission models.UserExamSubmission
	if err := db.WithContext(ctx).Preload("User").Preload("Answers").First(&submission, submissionID).Error; err != nil {
		return nil, err
	}
	return &submission, nil
}

func (r *submissionRepository) List(ctx context.Context, db *gorm.DB, examID uint, limit, offset int, search, order string, moodleSynced *bool) ([]models.UserExamSubmission, error) {
	var subs []models.UserExamSubmission
	query := db.WithContext(ctx).Preload("User").Preload("Answers").
		Where("exam_id = ?", examID)

	query = query.Joins("LEFT JOIN exam_user ON exam_user.id = user_exam_submission.user_id")

	if moodleSynced != nil {
		if *moodleSynced {
			query = query.Where("exam_user.moodle_id IS NOT NULL")
		} else {
			query = query.Where("exam_user.moodle_id IS NULL")
		}
	}

	if sanitized := strings.TrimSpace(search); sanitized != "" {
		like := fmt.Sprintf("%%%s%%", strings.ToLower(sanitized))
		query = query.Where(
			"LOWER(COALESCE(exam_user.name, '')) LIKE ? OR "+
				"LOWER(COALESCE(exam_user.surname, '')) LIKE ? OR "+
				"LOWER(COALESCE(exam_user.email, '')) LIKE ? OR "+
				"LOWER(COALESCE(exam_user.dni, '')) LIKE ?",
			like, like, like, like,
		)
	}

	if order != "" {
		query = query.Order(order)
	} else {
		query = query.Order("submitted_at DESC")
	}

	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	if err := query.Find(&subs).Error; err != nil {
		return nil, err
	}
	return subs, nil
}

func (r *submissionRepository) Count(ctx context.Context, db *gorm.DB, examID uint, moodleSynced *bool) (int64, error) {
	var total int64
	query := db.WithContext(ctx).Model(&models.UserExamSubmission{}).Where("exam_id = ?", examID)
	
	if moodleSynced != nil {
		query = query.Joins("LEFT JOIN exam_user ON exam_user.id = user_exam_submission.user_id")
		if *moodleSynced {
			query = query.Where("exam_user.moodle_id IS NOT NULL")
		} else {
			query = query.Where("exam_user.moodle_id IS NULL")
		}
	}
	
	err := query.Count(&total).Error
	return total, err
}

func (r *submissionRepository) Delete(ctx context.Context, db *gorm.DB, submissionID uint) error {
	// Atomic delete of submission and answers
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("submission_id = ?", submissionID).Delete(&models.UserAnswer{}).Error; err != nil {
			return err
		}
		if err := tx.Delete(&models.UserExamSubmission{}, submissionID).Error; err != nil {
			return err
		}
		return nil
	})
}

func (r *submissionRepository) DeleteByExamID(ctx context.Context, db *gorm.DB, examID uint) error {
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var submissionIDs []uint
		if err := tx.Model(&models.UserExamSubmission{}).
			Where("exam_id = ?", examID).
			Pluck("id", &submissionIDs).Error; err != nil {
			return err
		}

		if len(submissionIDs) > 0 {
			if err := tx.Where("submission_id IN (?)", submissionIDs).Delete(&models.UserAnswer{}).Error; err != nil {
				return err
			}
			if err := tx.Where("id IN (?)", submissionIDs).Delete(&models.UserExamSubmission{}).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *submissionRepository) GetAverageScore(ctx context.Context, db *gorm.DB, examID uint) (*float64, error) {
	var avg sql.NullFloat64
	if err := db.WithContext(ctx).Model(&models.UserExamSubmission{}).
		Select("AVG(score)").
		Where("exam_id = ? AND score IS NOT NULL", examID).
		Row().Scan(&avg); err != nil {
		return nil, err
	}
	if avg.Valid {
		return &avg.Float64, nil
	}
	return nil, nil
}

func (r *submissionRepository) Update(ctx context.Context, db *gorm.DB, submission *models.UserExamSubmission) error {
	return db.WithContext(ctx).Preload("User").Preload("Answers").Save(submission).Error
}

func (r *submissionRepository) SaveAnswer(ctx context.Context, db *gorm.DB, answer *models.UserAnswer) error {
	return db.WithContext(ctx).Save(answer).Error
}

func (r *submissionRepository) CreateAnswer(ctx context.Context, db *gorm.DB, answer *models.UserAnswer) error {
	return db.WithContext(ctx).Create(answer).Error
}
