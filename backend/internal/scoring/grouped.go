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
// PointsPerWrong (a fraction of one correct answer) per wrong answer, scaled by
// points_per_correct, floored at 0 and capped at the group's MaxScore.
// Only active, non-cancelled questions assigned to the group count. Total is the
// rounded sum of group scores. A group Passes when it has no minimum or its score
// reaches the minimum.
//
// When wrongBlockSize > 0 the per-group wrong penalty switches to block mode:
// every N wrong answers WITHIN THAT GROUP subtract 1 whole question's worth of
// credit (group.MaxScore/total), replacing PointsPerWrong. The block count is
// per group, so wrongs spread across groups don't accumulate.
//
// This is the single source of truth for grouped scoring: both the submit path
// and the batch recalc path call it, so they can never diverge.
func ComputeGrouped(groups []models.QuestionGroup, questions []models.Question, answers map[uint]string, wrongBlockSize float64) GroupedResult {
	res := GroupedResult{Groups: make([]GroupOutcome, 0, len(groups))}
	for _, g := range groups {
		correct, incorrect, total := tallyGroup(g, questions, answers)
		var score float64
		if total > 0 {
			ppc := g.MaxScore / float64(total)
			if wrongBlockSize > 0 {
				// block mode: subtract 1 whole question (ppc) per N wrongs in this group.
				score = (float64(correct) - math.Floor(float64(incorrect)/wrongBlockSize)) * ppc
				if score < 0 {
					score = 0
				}
				score = math.Round(score*100) / 100
			} else {
				// PointsPerWrong is a fraction of a correct answer (e.g. 0.25 = a
				// quarter), so scale it by ppc just like the credit. This matches the
				// flat legacy model — (correct − incorrect*penalty)/total*max — which
				// also deducts in question units before scaling.
				score = ComputeScore(correct, incorrect, total, Config{
					Mode:             "absolute",
					PointsPerCorrect: ppc,
					PointsPerWrong:   g.PointsPerWrong * ppc,
				})
			}
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

// tallyGroup counts correct / incorrect / total for one group, considering only
// active, non-cancelled questions assigned to it. Shared by every grouped scorer
// so the question-selection rules can never diverge.
func tallyGroup(g models.QuestionGroup, questions []models.Question, answers map[uint]string) (correct, incorrect, total int) {
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
	return correct, incorrect, total
}

// ComputeGroupedXunta scores a "Modo Xunta" grouped exam. Per group it derives the
// net pass rate p = clamp((correct − incorrect*PointsPerWrong) / total, 0, 1) and
// maps it to a grade with a piecewise-linear curve that pins the passing
// percentage to the passing grade:
//
//	p <= pp : grade = (p/pp) * ps
//	p >  pp : grade = ps + (p−pp)/(1−pp) * (MaxScore − ps)
//
// where pp = PassingPct/100 and ps = MinPassingScore. A group missing a usable
// pp/ps falls back to a single straight line p*MaxScore. A group Passes when its
// grade reaches MinPassingScore (i.e. p >= pp). Total is the rounded sum.
//
// All other exam scoring config (penalty type, block size, absolute/legacy mode)
// is intentionally ignored: Modo Xunta is self-contained per group.
func ComputeGroupedXunta(groups []models.QuestionGroup, questions []models.Question, answers map[uint]string) GroupedResult {
	res := GroupedResult{Groups: make([]GroupOutcome, 0, len(groups))}
	for _, g := range groups {
		correct, incorrect, total := tallyGroup(g, questions, answers)
		var score float64
		if total > 0 {
			net := float64(correct) - float64(incorrect)*g.PointsPerWrong
			p := net / float64(total)
			if p < 0 {
				p = 0
			}
			if p > 1 {
				p = 1
			}
			score = xuntaCurve(p, g)
			if score < 0 {
				score = 0
			}
			if score > g.MaxScore {
				score = g.MaxScore
			}
			score = math.Round(score*100) / 100
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

// xuntaCurve maps a pass rate p in [0,1] to a grade via the two-segment line that
// pins (pp, ps). When the group lacks a usable passing percentage or passing
// grade it degrades to the plain p*MaxScore line.
func xuntaCurve(p float64, g models.QuestionGroup) float64 {
	if g.PassingPct == nil || g.MinPassingScore == nil || *g.PassingPct <= 0 || *g.PassingPct >= 100 {
		return p * g.MaxScore
	}
	pp := *g.PassingPct / 100
	ps := *g.MinPassingScore
	if p <= pp {
		return (p / pp) * ps
	}
	return ps + (p-pp)/(1-pp)*(g.MaxScore-ps)
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
