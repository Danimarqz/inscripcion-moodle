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
	Count(ctx context.Context, db *gorm.DB, examIDs []uint, moodleSynced *bool, resultType *string) (int64, error)
	Delete(ctx context.Context, db *gorm.DB, submissionID uint) error
	DeleteByExamID(ctx context.Context, db *gorm.DB, examID uint) error
	GetAverageScore(ctx context.Context, db *gorm.DB, examID uint, moodleSynced *bool, resultType *string, group bool) (avg *float64, avgOfficial *float64, examIDs []uint, err error)
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
	// answers_json is intentionally excluded from the list payload; it's heavy
	// and lazy-loaded per submission via /admin/results/{id} when editing.
	query := db.WithContext(ctx).Preload("User").
		Omit("answers_json").
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

func (r *submissionRepository) Count(ctx context.Context, db *gorm.DB, examIDs []uint, moodleSynced *bool, resultType *string) (int64, error) {
	var total int64
	query := db.WithContext(ctx).Model(&models.UserExamSubmission{}).Where("exam_id IN ?", examIDs)

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

func (r *submissionRepository) GetAverageScore(ctx context.Context, db *gorm.DB, examID uint, moodleSynced *bool, resultType *string, group bool) (avg *float64, avgOfficial *float64, examIDs []uint, err error) {
	// Scope: just this exam, or every exam in its percentile group.
	examIDs = []uint{examID}
	if group {
		var exam models.Exam
		if e := db.WithContext(ctx).Select("percentile_group").First(&exam, examID).Error; e == nil && exam.PercentileGroup != nil {
			var ids []uint
			if e := db.WithContext(ctx).Model(&models.Exam{}).
				Where("percentile_group = ?", *exam.PercentileGroup).Pluck("id", &ids).Error; e == nil && len(ids) > 0 {
				examIDs = ids
			}
		}
	}

	var row struct {
		Sim      sql.NullFloat64
		Off      sql.NullFloat64
		OffCount int64
	}
	// Dedup officials to one score per (exam_id, user_id): a raw LEFT JOIN would
	// multiply submission rows when a user has several official rows (duplicate
	// masked DNIs / result types), skewing every aggregate.
	query := db.WithContext(ctx).Model(&models.UserExamSubmission{}).
		Joins(`LEFT JOIN (
			SELECT exam_id, user_id, MAX(score) AS score
			FROM exam_official_result
			WHERE score IS NOT NULL
			GROUP BY exam_id, user_id
		) o ON o.exam_id = user_exam_submission.exam_id AND o.user_id = user_exam_submission.user_id`).
		Where("user_exam_submission.exam_id IN ? AND user_exam_submission.score IS NOT NULL", examIDs)

	if moodleSynced != nil {
		query = query.Joins("LEFT JOIN exam_user ON exam_user.id = user_exam_submission.user_id")
		if *moodleSynced {
			query = query.Where("exam_user.moodle_id IS NOT NULL")
		} else {
			query = query.Where("exam_user.moodle_id IS NULL")
		}
	}

	if resultType != nil && *resultType != "" {
		query = query.Where("user_exam_submission.selected_result_type = ?", *resultType)
	}

	if err = query.Select(
		"AVG(user_exam_submission.score) AS sim, " +
			"AVG(COALESCE(o.score, user_exam_submission.score)) AS off, " +
			"SUM(CASE WHEN o.score IS NOT NULL THEN 1 ELSE 0 END) AS off_count",
	).Scan(&row).Error; err != nil {
		return nil, nil, examIDs, err
	}

	if row.Sim.Valid {
		avg = &row.Sim.Float64
	}
	// Only surface the official average when at least one official score exists.
	if row.OffCount > 0 && row.Off.Valid {
		avgOfficial = &row.Off.Float64
	}
	return avg, avgOfficial, examIDs, nil
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

// GetMeritsRanking ranks submissions by weighted (score + merits). Official
// scores/merits override the submission's own via COALESCE, applied
// automatically whenever a linked official result exists.
func (r *submissionRepository) GetMeritsRanking(ctx context.Context, db *gorm.DB, examID uint, submissionID uint, passingThreshold float64, examWeight float64, skipWeights bool) (position *int, total *int, err error) {
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

	// Single pass: rank + total via window functions. RANK() gives ties the same
	// position (= strictly-better count + 1, matching the old better-count math).
	windowSQL := "SELECT ranked.pos, ranked.total FROM (" +
		"SELECT s.id, RANK() OVER (ORDER BY " + scoreExpr + " DESC) AS pos, COUNT(*) OVER () AS total " +
		baseJoin + ") ranked WHERE ranked.id = ?"

	var rows []struct {
		Pos   int64
		Total int64
	}
	if err := db.WithContext(ctx).Raw(windowSQL, examID, passingThreshold, submissionID).Scan(&rows).Error; err != nil {
		return nil, nil, err
	}
	if len(rows) > 0 {
		pos := int(rows[0].Pos)
		total := int(rows[0].Total)
		return &pos, &total, nil
	}

	// Submission isn't in the qualifying set (below threshold / no merits): still
	// return the qualifying-set size, with no position.
	var totalCount int64
	if err := db.WithContext(ctx).Raw("SELECT COUNT(*) "+baseJoin, examID, passingThreshold).Scan(&totalCount).Error; err != nil {
		return nil, nil, err
	}
	totalInt := int(totalCount)
	return nil, &totalInt, nil
}
