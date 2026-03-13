package repository

import (
	"context"
	"fmt"

	"github.com/inscripcion-moodle/go-backend/internal/helpers"
	"github.com/inscripcion-moodle/go-backend/internal/models"
	"gorm.io/gorm"
)

type ExamRepository interface {
	FindExamByID(ctx context.Context, db *gorm.DB, examID uint) (*models.Exam, error)
	FindExamBySlug(ctx context.Context, db *gorm.DB, slug string) (*models.Exam, error)
	FindQuestionsByExamID(ctx context.Context, db *gorm.DB, examID uint) ([]models.Question, error)
	ListExams(ctx context.Context, db *gorm.DB) ([]models.Exam, error)
	CreateExam(ctx context.Context, db *gorm.DB, exam *models.Exam) error
	UpdateExam(ctx context.Context, db *gorm.DB, exam *models.Exam) error
	DeleteExam(ctx context.Context, db *gorm.DB, examID uint) error
	CountByName(ctx context.Context, db *gorm.DB, name string) (int64, error)
	RecalculateScores(ctx context.Context, db *gorm.DB, examID uint) error
	RecalculateScoresForSubmission(ctx context.Context, db *gorm.DB, examID uint, submissionID uint) error
	RecalculatePercentiles(ctx context.Context, db *gorm.DB, examID uint) error
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
	exam.Slug = helpers.CreateSlug(exam.Name)
	return &exam, nil
}

func (r *examRepository) FindExamBySlug(ctx context.Context, db *gorm.DB, slug string) (*models.Exam, error) {
	var exams []models.Exam
	// We fetch all active exams and search for the one that matches the slug.
	// This is safe because the number of exams is typically small.
	if err := db.WithContext(ctx).Where("is_active = ?", true).Find(&exams).Error; err != nil {
		return nil, err
	}

	for i := range exams {
		if helpers.CreateSlug(exams[i].Name) == slug {
			exams[i].Slug = helpers.CreateSlug(exams[i].Name)
			return &exams[i], nil
		}
	}

	return nil, gorm.ErrRecordNotFound
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
	for i := range exams {
		exams[i].Slug = helpers.CreateSlug(exams[i].Name)
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

func (r *examRepository) RecalculateScores(ctx context.Context, db *gorm.DB, examID uint) error {
	var exam models.Exam
	if err := db.WithContext(ctx).First(&exam, examID).Error; err != nil {
		return err
	}

	penalty := 0.0
	if exam.PenaltyValue != nil {
		penalty = *exam.PenaltyValue
	}

	maxScore := 100.0
	if exam.MaxScore != nil {
		maxScore = *exam.MaxScore
	}

	var totalQuestions int64
	if err := db.WithContext(ctx).Model(&models.Question{}).
		Where("exam_id = ? AND is_active = 1 AND is_cancelled = 0", examID).
		Count(&totalQuestions).Error; err != nil {
		return err
	}

	if totalQuestions == 0 {
		return nil // No questions, cannot calculate score
	}

	fmt.Printf("DEBUG: Recalculating scores for exam %d. Subtracts: %v, Penalty: %f, TotalQuestions: %d\n", examID, exam.SubtractsPoints, penalty, totalQuestions)

	const scoreSQL = `
UPDATE user_exam_submission AS u
JOIN (
    SELECT ua.submission_id,
           ROUND(
             SUM(
               CASE 
                 WHEN ua.answer = q.correct_option THEN 1.0 
                 WHEN ? AND ua.answer != '' THEN -? 
                 ELSE 0.0 
               END
             ) / ? * ?, 
           2) AS base_score
    FROM user_answer ua
    INNER JOIN question q ON q.id = ua.question_id 
    WHERE q.exam_id = ?
      AND q.is_active = 1
      AND q.is_cancelled = 0
    GROUP BY ua.submission_id
) AS t ON u.id = t.submission_id
SET u.score = t.base_score
WHERE u.exam_id = ?`

	if err := db.WithContext(ctx).Exec(scoreSQL, exam.SubtractsPoints, penalty, float64(totalQuestions), maxScore, examID, examID).Error; err != nil {
		fmt.Printf("ERROR: RecalculateScores SQL failed: %v\n", err)
		return err
	}

	return r.RecalculatePercentiles(ctx, db, examID)
}

func (r *examRepository) RecalculatePercentiles(ctx context.Context, db *gorm.DB, examID uint) error {
	const percentileSQL = `
UPDATE user_exam_submission AS u
JOIN (
    SELECT
        id,
        ROUND(CUME_DIST() OVER (ORDER BY score ASC) * 100, 2) AS pct
    FROM user_exam_submission
    WHERE exam_id = ? AND score IS NOT NULL
) ranked ON ranked.id = u.id
SET u.percentile = ranked.pct
WHERE u.exam_id = ?`

	if err := db.WithContext(ctx).Exec(percentileSQL, examID, examID).Error; err != nil {
		fmt.Printf("ERROR: RecalculatePercentiles SQL failed: %v\n", err)
		return err
	}
	return nil
}

func (r *examRepository) RecalculateScoresForSubmission(ctx context.Context, db *gorm.DB, examID uint, submissionID uint) error {
	var exam models.Exam
	if err := db.WithContext(ctx).First(&exam, examID).Error; err != nil {
		return err
	}

	penalty := 0.0
	if exam.PenaltyValue != nil {
		penalty = *exam.PenaltyValue
	}

	maxScore := 100.0
	if exam.MaxScore != nil {
		maxScore = *exam.MaxScore
	}

	var totalQuestions int64
	if err := db.WithContext(ctx).Model(&models.Question{}).
		Where("exam_id = ? AND is_active = 1 AND is_cancelled = 0", examID).
		Count(&totalQuestions).Error; err != nil {
		return err
	}

	if totalQuestions == 0 {
		return nil // No questions, cannot calculate score
	}

	const scoreSQL = `
UPDATE user_exam_submission AS u
JOIN (
    SELECT ua.submission_id,
           ROUND(
             SUM(
               CASE 
                 WHEN ua.answer = q.correct_option THEN 1.0 
                 WHEN ? AND ua.answer != '' THEN -? 
                 ELSE 0.0 
               END
             ) / ? * ?, 
           2) AS base_score
    FROM user_answer ua
    INNER JOIN question q ON q.id = ua.question_id 
    WHERE q.exam_id = ? AND ua.submission_id = ?
      AND q.is_active = 1
      AND q.is_cancelled = 0
    GROUP BY ua.submission_id
) AS t ON u.id = t.submission_id
SET u.score = t.base_score
WHERE u.exam_id = ? AND u.id = ?`

	if err := db.WithContext(ctx).Exec(scoreSQL, exam.SubtractsPoints, penalty, float64(totalQuestions), maxScore, examID, submissionID, examID, submissionID).Error; err != nil {
		fmt.Printf("ERROR: RecalculateScoresForSubmission SQL failed: %v\n", err)
		return err
	}

	return r.RecalculatePercentiles(ctx, db, examID)
}
