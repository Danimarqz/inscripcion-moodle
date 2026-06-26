package repository

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strings"

	"github.com/inscripcion-moodle/go-backend/internal/helpers"
	"github.com/inscripcion-moodle/go-backend/internal/models"
	"github.com/inscripcion-moodle/go-backend/internal/scoring"
	"gorm.io/gorm"
)

type ExamRepository interface {
	FindExamByID(ctx context.Context, db *gorm.DB, examID uint) (*models.Exam, error)
	FindExamByIDLite(ctx context.Context, db *gorm.DB, examID uint) (*models.Exam, error)
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
	SetPercentileGroup(ctx context.Context, db *gorm.DB, examID uint, memberIDs []uint) ([]uint, error)
	GetTop10AverageScore(ctx context.Context, db *gorm.DB, examID uint) (*float64, error)
	CountActiveQuestions(ctx context.Context, db *gorm.DB, examID uint) (int, error)
}

type examRepository struct{}

func NewExamRepository() ExamRepository {
	return &examRepository{}
}

func (r *examRepository) FindExamByID(ctx context.Context, db *gorm.DB, examID uint) (*models.Exam, error) {
	var exam models.Exam
	if err := db.WithContext(ctx).Preload("Questions").Preload("Groups").First(&exam, examID).Error; err != nil {
		return nil, err
	}
	exam.Slug = helpers.CreateSlug(exam.Name)
	return &exam, nil
}

// FindExamByIDLite loads the exam without its questions. Questions are fetched
// on demand via FindQuestionsByExamID, so the exam-config read stays small.
func (r *examRepository) FindExamByIDLite(ctx context.Context, db *gorm.DB, examID uint) (*models.Exam, error) {
	var exam models.Exam
	if err := db.WithContext(ctx).First(&exam, examID).Error; err != nil {
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
	// Questions are intentionally not preloaded: the exam list never uses them
	// and loading every question of every exam bloats this frequent endpoint.
	if err := db.WithContext(ctx).Find(&exams).Error; err != nil {
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

	cfg := scoring.ConfigFromExamFields(exam.ScoringMode, exam.SubtractsPoints, exam.PenaltyValue, exam.MaxScore, exam.PointsPerCorrect, exam.PointsPerWrong)

	var questions []models.Question
	if err := db.WithContext(ctx).Where("exam_id = ? AND is_active = 1 AND is_cancelled = 0", examID).
		Find(&questions).Error; err != nil {
		return err
	}
	if len(questions) == 0 {
		return nil
	}

	// Grouped exams recalc via scoring.ComputeGrouped (same source as the submit
	// path); flat exams use the single-config calculateScore.
	var groups []models.QuestionGroup
	if err := db.WithContext(ctx).Where("exam_id = ?", examID).Find(&groups).Error; err != nil {
		return err
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
		var score float64
		if len(groups) > 0 {
			score = scoring.ComputeGrouped(groups, questions, map[uint]string(*sub.AnswersData)).Total
		} else {
			score = calculateScore(questions, map[uint]string(*sub.AnswersData), cfg)
		}
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
	// Resolve the pool: if this exam belongs to a percentile group, percentiles
	// are computed over every exam sharing that group; otherwise just this exam.
	examIDs := []uint{examID}
	var exam models.Exam
	if err := db.WithContext(ctx).Select("percentile_group").First(&exam, examID).Error; err == nil && exam.PercentileGroup != nil {
		var ids []uint
		if err := db.WithContext(ctx).Model(&models.Exam{}).
			Where("percentile_group = ?", *exam.PercentileGroup).Pluck("id", &ids).Error; err == nil && len(ids) > 0 {
			examIDs = ids
		}
	}

	// Score per row = official score if present, else the submission score. This
	// makes official scores apply automatically without a per-exam flag.
	percentileSQL := `
UPDATE user_exam_submission AS u
JOIN (
    SELECT
        s.id,
        ROUND(CUME_DIST() OVER (ORDER BY COALESCE(o.score, s.score) ASC) * 100, 2) AS pct
    FROM user_exam_submission s
    LEFT JOIN exam_official_result o
        ON o.exam_id = s.exam_id AND o.user_id = s.user_id AND o.score IS NOT NULL
    WHERE s.exam_id IN (?) AND (s.score IS NOT NULL OR o.score IS NOT NULL)
) ranked ON ranked.id = u.id
SET u.percentile = ranked.pct
WHERE u.exam_id IN (?)`

	if err := db.WithContext(ctx).Exec(percentileSQL, examIDs, examIDs).Error; err != nil {
		log.Printf("ERROR: RecalculatePercentiles SQL failed: %v", err)
		return err
	}
	return nil
}

// SetPercentileGroup makes examID's percentile group exactly {examID} ∪ memberIDs.
// The relationship is symmetric: every member ends up sharing one group id
// (= the smallest member id). Exams that leave examID's previous group are
// detached, and any group left with a single member is dissolved (set NULL).
// Returns the ids of every exam whose group membership may have changed, so the
// caller can recalculate percentiles for each affected pool.
func (r *examRepository) SetPercentileGroup(ctx context.Context, db *gorm.DB, examID uint, memberIDs []uint) ([]uint, error) {
	set := map[uint]bool{examID: true}
	for _, id := range memberIDs {
		set[id] = true
	}
	members := make([]uint, 0, len(set))
	for id := range set {
		members = append(members, id)
	}

	affected := map[uint]bool{examID: true}
	for _, id := range members {
		affected[id] = true
	}

	var exam models.Exam
	db.WithContext(ctx).Select("percentile_group").First(&exam, examID)

	err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Detach exams that were in examID's old group but are no longer members.
		if exam.PercentileGroup != nil {
			var old []uint
			tx.Model(&models.Exam{}).Where("percentile_group = ?", *exam.PercentileGroup).Pluck("id", &old)
			for _, id := range old {
				affected[id] = true
			}
			if err := tx.Model(&models.Exam{}).
				Where("percentile_group = ? AND id NOT IN ?", *exam.PercentileGroup, members).
				Update("percentile_group", nil).Error; err != nil {
				return err
			}
		}

		if len(members) < 2 {
			// Nothing to associate: detach examID.
			if err := tx.Model(&models.Exam{}).Where("id = ?", examID).
				Update("percentile_group", nil).Error; err != nil {
				return err
			}
		} else {
			groupID := members[0]
			for _, id := range members {
				if id < groupID {
					groupID = id
				}
			}
			if err := tx.Model(&models.Exam{}).Where("id IN ?", members).
				Update("percentile_group", groupID).Error; err != nil {
				return err
			}
		}

		// Dissolve any group left with a single member.
		return tx.Exec(`UPDATE exam SET percentile_group = NULL WHERE percentile_group IN (
    SELECT g FROM (
        SELECT percentile_group AS g FROM exam
        WHERE percentile_group IS NOT NULL GROUP BY percentile_group HAVING COUNT(*) = 1
    ) t)`).Error
	})
	if err != nil {
		return nil, err
	}

	ids := make([]uint, 0, len(affected))
	for id := range affected {
		ids = append(ids, id)
	}
	return ids, nil
}

func (r *examRepository) RecalculateScoresForSubmission(ctx context.Context, db *gorm.DB, examID uint, submissionID uint) error {
	var exam models.Exam
	if err := db.WithContext(ctx).First(&exam, examID).Error; err != nil {
		return err
	}

	cfg := scoring.ConfigFromExamFields(exam.ScoringMode, exam.SubtractsPoints, exam.PenaltyValue, exam.MaxScore, exam.PointsPerCorrect, exam.PointsPerWrong)

	var questions []models.Question
	if err := db.WithContext(ctx).Where("exam_id = ? AND is_active = 1 AND is_cancelled = 0", examID).
		Find(&questions).Error; err != nil {
		return err
	}
	if len(questions) == 0 {
		return nil
	}

	var groups []models.QuestionGroup
	if err := db.WithContext(ctx).Where("exam_id = ?", examID).Find(&groups).Error; err != nil {
		return err
	}

	var submission models.UserExamSubmission
	if err := db.WithContext(ctx).First(&submission, submissionID).Error; err != nil {
		return err
	}
	if submission.AnswersData == nil {
		return r.RecalculatePercentiles(ctx, db, examID)
	}

	var score float64
	if len(groups) > 0 {
		score = scoring.ComputeGrouped(groups, questions, map[uint]string(*submission.AnswersData)).Total
	} else {
		score = calculateScore(questions, map[uint]string(*submission.AnswersData), cfg)
	}
	if err := db.WithContext(ctx).Model(&models.UserExamSubmission{}).
		Where("id = ?", submissionID).
		Update("score", score).Error; err != nil {
		return err
	}

	return r.RecalculatePercentiles(ctx, db, examID)
}

func (r *examRepository) CountActiveQuestions(ctx context.Context, db *gorm.DB, examID uint) (int, error) {
	var count int64
	err := db.WithContext(ctx).Model(&models.Question{}).
		Where("exam_id = ? AND is_active = 1 AND is_cancelled = 0", examID).
		Count(&count).Error
	return int(count), err
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

func calculateScore(questions []models.Question, answers map[uint]string, cfg scoring.Config) float64 {
	total, correct, incorrect := 0, 0, 0
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
			correct++
		} else {
			incorrect++
		}
	}
	return scoring.ComputeScore(correct, incorrect, total, cfg)
}
