package scoring

import (
	"math"
	"strings"

	"github.com/inscripcion-moodle/go-backend/internal/models"
)

// GroupOutcome is one question group's scored result within a grouped exam.
type GroupOutcome struct {
	GroupID         uint     `json:"group_id"`
	Name            string   `json:"name"`
	Score           float64  `json:"score"`
	MaxScore        float64  `json:"max_score"`
	Correct         int      `json:"correct"`
	Incorrect       int      `json:"incorrect"`
	Total           int      `json:"total"`
	Eliminatory     bool     `json:"eliminatory"`
	MinPassingScore *float64 `json:"min_passing_score,omitempty"`
	Passed          bool     `json:"passed"`
}

// GroupedResult is the full per-group + total outcome of a grouped submission.
type GroupedResult struct {
	Total  float64
	Groups []GroupOutcome
}

// ComputeGrouped scores each group with the absolute model:
// points_per_correct = group.MaxScore / (active questions in that group), minus
// PointsPerWrong per wrong answer, floored at 0 and capped at the group's MaxScore.
// Only active, non-cancelled questions assigned to the group count. Total is the
// rounded sum of group scores. A group Passes when it has no minimum or its score
// reaches the minimum.
//
// This is the single source of truth for grouped scoring: both the submit path
// and the batch recalc path call it, so they can never diverge.
func ComputeGrouped(groups []models.QuestionGroup, questions []models.Question, answers map[uint]string) GroupedResult {
	res := GroupedResult{Groups: make([]GroupOutcome, 0, len(groups))}
	for _, g := range groups {
		correct, incorrect, total := 0, 0, 0
		for _, q := range questions {
			if q.GroupID == nil || *q.GroupID != g.ID || !q.IsActive || q.IsCancelled {
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
		var score float64
		if total > 0 {
			ppc := g.MaxScore / float64(total)
			score = ComputeScore(correct, incorrect, total, Config{
				Mode:             "absolute",
				PointsPerCorrect: ppc,
				PointsPerWrong:   g.PointsPerWrong,
			})
			if score > g.MaxScore {
				score = g.MaxScore
			}
		}
		passed := g.MinPassingScore == nil || score >= *g.MinPassingScore
		res.Groups = append(res.Groups, GroupOutcome{
			GroupID:         g.ID,
			Name:            g.Name,
			Score:           score,
			MaxScore:        g.MaxScore,
			Correct:         correct,
			Incorrect:       incorrect,
			Total:           total,
			Eliminatory:     g.Eliminatory,
			MinPassingScore: g.MinPassingScore,
			Passed:          passed,
		})
		res.Total += score
	}
	res.Total = math.Round(res.Total*100) / 100
	return res
}

// AllEliminatoryPassed reports whether every eliminatory group met its minimum.
// An exam with no eliminatory groups trivially passes this gate.
func (r GroupedResult) AllEliminatoryPassed() bool {
	for _, g := range r.Groups {
		if g.Eliminatory && !g.Passed {
			return false
		}
	}
	return true
}
