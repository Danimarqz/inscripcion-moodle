package repository

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"math"
	"strings"

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
	GetTop10AverageScore(ctx context.Context, db *gorm.DB, examID uint) (*float64, error)
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

	var questions []models.Question
	if err := db.WithContext(ctx).Where("exam_id = ? AND is_active = 1 AND is_cancelled = 0", examID).
		Find(&questions).Error; err != nil {
		return err
	}
	if len(questions) == 0 {
		return nil
	}

	var submissions []models.UserExamSubmission
	if err := db.WithContext(ctx).Where("exam_id = ?", examID).Find(&submissions).Error; err != nil {
		return err
	}

	// Calculate scores in Go and batch update
	type scoreUpdate struct {
		id    uint
		score float64
	}
	updates := make([]scoreUpdate, 0, len(submissions))
	for _, sub := range submissions {
		if sub.AnswersData == nil {
			continue
		}
		score := calculateScore(questions, map[uint]string(*sub.AnswersData), exam.SubtractsPoints, penalty, maxScore)
		updates = append(updates, scoreUpdate{id: sub.ID, score: score})
	}

	if len(updates) > 0 {
		var sb strings.Builder
		sb.WriteString("UPDATE user_exam_submission SET score = CASE id ")
		ids := make([]uint, 0, len(updates))
		for _, u := range updates {
			fmt.Fprintf(&sb, "WHEN %d THEN %.2f ", u.id, u.score)
			ids = append(ids, u.id)
		}
		sb.WriteString("END WHERE id IN (?)")
		if err := db.WithContext(ctx).Exec(sb.String(), ids).Error; err != nil {
			log.Printf("ERROR: RecalculateScores batch update failed: %v", err)
			return err
		}
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
		log.Printf("ERROR: RecalculatePercentiles SQL failed: %v", err)
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

	var questions []models.Question
	if err := db.WithContext(ctx).Where("exam_id = ? AND is_active = 1 AND is_cancelled = 0", examID).
		Find(&questions).Error; err != nil {
		return err
	}
	if len(questions) == 0 {
		return nil
	}

	var submission models.UserExamSubmission
	if err := db.WithContext(ctx).First(&submission, submissionID).Error; err != nil {
		return err
	}
	if submission.AnswersData == nil {
		return r.RecalculatePercentiles(ctx, db, examID)
	}

	score := calculateScore(questions, map[uint]string(*submission.AnswersData), exam.SubtractsPoints, penalty, maxScore)
	if err := db.WithContext(ctx).Model(&models.UserExamSubmission{}).
		Where("id = ?", submissionID).
		Update("score", score).Error; err != nil {
		return err
	}

	return r.RecalculatePercentiles(ctx, db, examID)
}

func (r *examRepository) GetTop10AverageScore(ctx context.Context, db *gorm.DB, examID uint) (*float64, error) {
	var avg sql.NullFloat64
	subquery := `SELECT AVG(score) FROM (SELECT score FROM user_exam_submission WHERE exam_id = ? AND score IS NOT NULL ORDER BY score DESC LIMIT 10) t`
	if err := db.WithContext(ctx).Raw(subquery, examID).Row().Scan(&avg); err != nil {
		return nil, err
	}
	if avg.Valid {
		return &avg.Float64, nil
	}
	return nil, nil
}

func calculateScore(questions []models.Question, answers map[uint]string, subtracts bool, penalty, maxScore float64) float64 {
	total := 0
	netCorrect := 0.0
	for _, q := range questions {
		if !q.IsActive || q.IsCancelled {
			continue
		}
		total++
		selected := strings.ToUpper(strings.TrimSpace(answers[q.ID]))
		if selected == "" {
			continue
		}
		if strings.EqualFold(selected, strings.TrimSpace(q.CorrectOption)) {
			netCorrect++
		} else if subtracts && penalty > 0 {
			netCorrect -= penalty
		}
	}
	if total == 0 {
		return 0
	}
	return math.Round(netCorrect/float64(total)*maxScore*100) / 100
}
