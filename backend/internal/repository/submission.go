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
	ExistsByStudentExam(ctx context.Context, db *gorm.DB, email, dni string, examID uint) (bool, error)
	List(ctx context.Context, db *gorm.DB, examID uint, limit, offset int, search, order string, moodleSynced *bool, resultType *string) ([]models.UserExamSubmission, error)
	Count(ctx context.Context, db *gorm.DB, examID uint, moodleSynced *bool, resultType *string) (int64, error)
	Delete(ctx context.Context, db *gorm.DB, submissionID uint) error
	DeleteByExamID(ctx context.Context, db *gorm.DB, examID uint) error
	GetAverageScore(ctx context.Context, db *gorm.DB, examID uint, moodleSynced *bool, resultType *string) (*float64, error)
	Update(ctx context.Context, db *gorm.DB, submission *models.UserExamSubmission) error
	SaveAnswer(ctx context.Context, db *gorm.DB, answer *models.UserAnswer) error   // Phase 1: dual-write
	CreateAnswer(ctx context.Context, db *gorm.DB, answer *models.UserAnswer) error // Phase 1: dual-write
	GetMeritsRanking(ctx context.Context, db *gorm.DB, examID uint, submissionID uint, passingThreshold float64, examWeight float64, skipWeights bool) (position *int, total *int, err error)
}

type submissionRepository struct{}

func NewSubmissionRepository() SubmissionRepository {
	return &submissionRepository{}
}

func (r *submissionRepository) FindByID(ctx context.Context, db *gorm.DB, submissionID uint) (*models.UserExamSubmission, error) {
	var submission models.UserExamSubmission
	if err := db.WithContext(ctx).Preload("User").First(&submission, submissionID).Error; err != nil {
		return nil, err
	}
	return &submission, nil
}

// ExistsByStudentExam reports whether a submission exists for the given student
// (matched by lowercased email + normalized DNI) and exam.
func (r *submissionRepository) ExistsByStudentExam(ctx context.Context, db *gorm.DB, email, dni string, examID uint) (bool, error) {
	var count int64
	err := db.WithContext(ctx).
		Model(&models.UserExamSubmission{}).
		Joins("JOIN exam_user ON exam_user.id = user_exam_submission.user_id").
		Where("LOWER(exam_user.email) = ? AND exam_user.dni = ? AND user_exam_submission.exam_id = ?",
			email, dni, examID).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *submissionRepository) List(ctx context.Context, db *gorm.DB, examID uint, limit, offset int, search, order string, moodleSynced *bool, resultType *string) ([]models.UserExamSubmission, error) {
	var subs []models.UserExamSubmission
	query := db.WithContext(ctx).Preload("User").
		Where("exam_id = ?", examID)

	query = query.Joins("LEFT JOIN exam_user ON exam_user.id = user_exam_submission.user_id")

	if moodleSynced != nil {
		if *moodleSynced {
			query = query.Where("exam_user.moodle_id IS NOT NULL")
		} else {
			query = query.Where("exam_user.moodle_id IS NULL")
		}
	}

	if resultType != nil && *resultType != "" {
		query = query.Where("user_exam_submission.selected_result_type = ?", *resultType)
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

func (r *submissionRepository) Count(ctx context.Context, db *gorm.DB, examID uint, moodleSynced *bool, resultType *string) (int64, error) {
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

	if resultType != nil && *resultType != "" {
		query = query.Where("selected_result_type = ?", *resultType)
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

func (r *submissionRepository) GetAverageScore(ctx context.Context, db *gorm.DB, examID uint, moodleSynced *bool, resultType *string) (*float64, error) {
	var avg sql.NullFloat64
	query := db.WithContext(ctx).Model(&models.UserExamSubmission{}).Where("exam_id = ? AND score IS NOT NULL", examID)

	if moodleSynced != nil {
		query = query.Joins("LEFT JOIN exam_user ON exam_user.id = user_exam_submission.user_id")
		if *moodleSynced {
			query = query.Where("exam_user.moodle_id IS NOT NULL")
		} else {
			query = query.Where("exam_user.moodle_id IS NULL")
		}
	}

	if resultType != nil && *resultType != "" {
		query = query.Where("selected_result_type = ?", *resultType)
	}

	if err := query.Select("AVG(score)").Row().Scan(&avg); err != nil {
		return nil, err
	}
	if avg.Valid {
		return &avg.Float64, nil
	}
	return nil, nil
}

func (r *submissionRepository) Update(ctx context.Context, db *gorm.DB, submission *models.UserExamSubmission) error {
	return db.WithContext(ctx).Save(submission).Error
}

func (r *submissionRepository) SaveAnswer(ctx context.Context, db *gorm.DB, answer *models.UserAnswer) error {
	return db.WithContext(ctx).Save(answer).Error
}

func (r *submissionRepository) CreateAnswer(ctx context.Context, db *gorm.DB, answer *models.UserAnswer) error {
	return db.WithContext(ctx).Create(answer).Error
}

func (r *submissionRepository) GetMeritsRanking(ctx context.Context, db *gorm.DB, examID uint, submissionID uint, passingThreshold float64, examWeight float64, skipWeights bool) (position *int, total *int, err error) {
	// Check if official score override is enabled
	var exam models.Exam
	useOfficial := false
	if err := db.WithContext(ctx).Select("use_official_scores").First(&exam, examID).Error; err == nil {
		useOfficial = exam.UseOfficialScores
	}

	if useOfficial {
		return r.getMeritsRankingWithOfficialScores(ctx, db, examID, submissionID, passingThreshold, examWeight, skipWeights)
	}

	var totalCount int64
	if err := db.WithContext(ctx).Model(&models.UserExamSubmission{}).
		Where("exam_id = ? AND score >= ? AND merits IS NOT NULL", examID, passingThreshold).
		Count(&totalCount).Error; err != nil {
		return nil, nil, err
	}
	totalInt := int(totalCount)

	var scoreExpr string
	var scoreArgs []any
	if skipWeights {
		scoreExpr = "(score + merits)"
	} else {
		meritsWeight := 1 - examWeight
		scoreExpr = "(score * ?) + (merits * ?)"
		scoreArgs = []any{examWeight, meritsWeight}
	}

	subquery := db.WithContext(ctx).Model(&models.UserExamSubmission{}).
		Select(scoreExpr, scoreArgs...).
		Where("id = ?", submissionID)

	betterArgs := []any{examID, passingThreshold}
	betterArgs = append(betterArgs, scoreArgs...)
	var betterCount int64
	if err := db.WithContext(ctx).Model(&models.UserExamSubmission{}).
		Where("exam_id = ? AND score >= ? AND merits IS NOT NULL AND "+scoreExpr+" > (?)", append(betterArgs, subquery)...).
		Count(&betterCount).Error; err != nil {
		return nil, nil, err
	}
	pos := int(betterCount) + 1
	return &pos, &totalInt, nil
}

func (r *submissionRepository) getMeritsRankingWithOfficialScores(ctx context.Context, db *gorm.DB, examID uint, submissionID uint, passingThreshold float64, examWeight float64, skipWeights bool) (position *int, total *int, err error) {
	var scoreExpr string
	if skipWeights {
		scoreExpr = "(COALESCE(o.score, s.score) + COALESCE(o.merits, s.merits))"
	} else {
		meritsWeight := 1 - examWeight
		scoreExpr = fmt.Sprintf("(COALESCE(o.score, s.score) * %f + COALESCE(o.merits, s.merits) * %f)", examWeight, meritsWeight)
	}

	baseJoin := `
		FROM user_exam_submission s
		LEFT JOIN exam_official_result o
			ON o.exam_id = s.exam_id AND o.user_id = s.user_id
		WHERE s.exam_id = ?
			AND COALESCE(o.score, s.score) >= ?
			AND COALESCE(o.merits, s.merits) IS NOT NULL`

	// Total count
	var totalCount int64
	countSQL := "SELECT COUNT(*) " + baseJoin
	if err := db.WithContext(ctx).Raw(countSQL, examID, passingThreshold).Scan(&totalCount).Error; err != nil {
		return nil, nil, err
	}
	totalInt := int(totalCount)

	// Get current submission's weighted score
	var myScore sql.NullFloat64
	myScoreSQL := fmt.Sprintf("SELECT %s %s AND s.id = ?", scoreExpr, baseJoin)
	if err := db.WithContext(ctx).Raw(myScoreSQL, examID, passingThreshold, submissionID).Scan(&myScore).Error; err != nil {
		return nil, nil, err
	}

	if !myScore.Valid {
		return nil, &totalInt, nil
	}

	// Count submissions with a better score
	betterSQL := fmt.Sprintf("SELECT COUNT(*) %s AND %s > ?", baseJoin, scoreExpr)
	var betterCount int64
	if err := db.WithContext(ctx).Raw(betterSQL, examID, passingThreshold, myScore.Float64).Scan(&betterCount).Error; err != nil {
		return nil, nil, err
	}

	pos := int(betterCount) + 1
	return &pos, &totalInt, nil
}
