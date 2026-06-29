package exam

import (
	"maps"
	"testing"

	"github.com/inscripcion-moodle/go-backend/internal/models"
	"github.com/stretchr/testify/assert"
)

// fptr is a tiny helper for *float64 group minimums.
func fptr(v float64) *float64 { return &v }

// buildXuntaExam builds the two-group Xunta exam: Teórico (60 pts / 80 preguntas /
// -0.25 / mín 30 / eliminatorio) y Práctico (40 pts / 40 preguntas / -0.25 / mín 20
// / eliminatorio). Returns groups + questions; group 1 = teórico, group 2 = práctico.
func buildXuntaExam() ([]models.QuestionGroup, []models.Question) {
	g1, g2 := uint(1), uint(2)
	groups := []models.QuestionGroup{
		{ID: 1, Name: "Teórico", Position: 0, MaxScore: 60, PointsPerWrong: 0.25, MinPassingScore: fptr(30), Eliminatory: true},
		{ID: 2, Name: "Práctico", Position: 1, MaxScore: 40, PointsPerWrong: 0.25, MinPassingScore: fptr(20), Eliminatory: true},
	}
	var questions []models.Question
	for i := 1; i <= 80; i++ {
		questions = append(questions, models.Question{ID: uint(i), GroupID: &g1, IsActive: true, CorrectOption: "A"})
	}
	for i := 81; i <= 120; i++ {
		questions = append(questions, models.Question{ID: uint(i), GroupID: &g2, IsActive: true, CorrectOption: "A"})
	}
	return groups, questions
}

func answerAll(questions []models.Question, correct, wrong int, groupID uint) map[uint]string {
	// Returns answers for `correct` rights and `wrong` wrongs within groupID; rest blank.
	answers := map[uint]string{}
	c, w := correct, wrong
	for _, q := range questions {
		if q.GroupID == nil || *q.GroupID != groupID {
			continue
		}
		if c > 0 {
			answers[q.ID] = "A" // correct
			c--
		} else if w > 0 {
			answers[q.ID] = "B" // wrong
			w--
		}
	}
	return answers
}

func merge(a, b map[uint]string) map[uint]string {
	out := map[uint]string{}
	maps.Copy(out, a)
	maps.Copy(out, b)
	return out
}

func TestGroupedBreakdown_Xunta_AllCorrect(t *testing.T) {
	groups, questions := buildXuntaExam()
	answers := merge(answerAll(questions, 80, 0, 1), answerAll(questions, 40, 0, 2))

	bd, err := CalculateGroupedBreakdown(groups, questions, answers, 0)
	assert.NoError(t, err)
	assert.Equal(t, 100.0, bd.Score) // 60 + 40
	assert.Equal(t, 120, bd.TotalQuestions)
	assert.Equal(t, 120, bd.CorrectAnswers)
	assert.Len(t, bd.Groups, 2)
	assert.Equal(t, 60.0, bd.Groups[0].Score)
	assert.Equal(t, 40.0, bd.Groups[1].Score)
	assert.True(t, bd.Groups[0].Passed)
	assert.True(t, bd.Groups[1].Passed)
}

func TestGroupedBreakdown_Xunta_TheoryFailsEliminatory(t *testing.T) {
	groups, questions := buildXuntaExam()
	// Teórico: ppc = 60/80 = 0.75. Penalización escalada = 0.25*0.75 = 0.1875 por fallo.
	// 44 correct, 36 wrong => 44*0.75 - 36*0.1875 = 33 - 6.75 = 26.25 < 30 => falla eliminatorio.
	answers := merge(answerAll(questions, 44, 36, 1), answerAll(questions, 40, 0, 2))

	bd, err := CalculateGroupedBreakdown(groups, questions, answers, 0)
	assert.NoError(t, err)
	assert.InDelta(t, 26.25, bd.Groups[0].Score, 0.001)
	assert.False(t, bd.Groups[0].Passed)
	assert.True(t, bd.Groups[1].Passed)
}

func TestGroupedBreakdown_BlockPenalty_PerGroup(t *testing.T) {
	// Block penalty N=4: wrongs counted per group, never across groups.
	// Teórico ppc=60/80=0.75, Práctico ppc=40/40=1.0.
	groups, questions := buildXuntaExam()

	// 3 wrong in Teórico + 1 wrong in Práctico => neither group reaches a full
	// block of 4, so NO deduction in either group.
	spread := merge(answerAll(questions, 77, 3, 1), answerAll(questions, 39, 1, 2))
	bd, err := CalculateGroupedBreakdown(groups, questions, spread, 4)
	assert.NoError(t, err)
	assert.InDelta(t, 57.75, bd.Groups[0].Score, 0.001) // 77 * 0.75, no deduction
	assert.InDelta(t, 39.0, bd.Groups[1].Score, 0.001)  // 39 * 1.0, no deduction

	// 4 wrong in Teórico => exactly 1 whole question off in THAT group only.
	concentrated := merge(answerAll(questions, 76, 4, 1), answerAll(questions, 40, 0, 2))
	bd, err = CalculateGroupedBreakdown(groups, questions, concentrated, 4)
	assert.NoError(t, err)
	assert.InDelta(t, 56.25, bd.Groups[0].Score, 0.001) // (76 - 1) * 0.75
	assert.InDelta(t, 40.0, bd.Groups[1].Score, 0.001)  // untouched
}

// Modo Xunta final-score model: per-group card = netas scaled to valoración
// (groupScore = p*max). Total = sum of the official linear scaling applied to that
// scaled score, anchors (cut → max/2) and (total → max) in count units. Cuts fixed:
// Teórico 24, Práctico 18. A perfect group does NOT reach max (spec quirk).

func TestGroupedXunta_NereaCase(t *testing.T) {
	groups, questions := buildXuntaExam()
	// Teórico 49 aciertos / 28 fallos => netas 42 => groupScore 42/80*60 = 31.5.
	// Práctico 30 / 10 => netas 27.5 => groupScore 27.5.
	answers := merge(answerAll(questions, 49, 28, 1), answerAll(questions, 30, 10, 2))
	bd, err := CalculateGroupedXuntaBreakdown(groups, questions, answers)
	assert.NoError(t, err)
	assert.InDelta(t, 31.5, bd.Groups[0].Score, 0.001)
	assert.InDelta(t, 27.5, bd.Groups[1].Score, 0.001)
	assert.True(t, bd.Groups[0].Passed) // 31.5 >= 24
	assert.True(t, bd.Groups[1].Passed) // 27.5 >= 18
	// Per group rounded then summed: 34.02 + 28.64 = 62.66.
	assert.InDelta(t, 62.66, bd.Score, 0.001)
}

func TestGroupedXunta_Anchors(t *testing.T) {
	groups, questions := buildXuntaExam()
	// groupScore at the cut => contributes max/2. Teórico cut 24 needs groupScore 24
	// => netas 32 (32/80*60=24). Práctico cut 18 => groupScore 18 => netas 18.
	atCut := merge(answerAll(questions, 32, 0, 1), answerAll(questions, 18, 0, 2))
	bd, err := CalculateGroupedXuntaBreakdown(groups, questions, atCut)
	assert.NoError(t, err)
	assert.InDelta(t, 24.0, bd.Groups[0].Score, 0.001)
	assert.InDelta(t, 18.0, bd.Groups[1].Score, 0.001)
	assert.InDelta(t, 30.0+20.0, bd.Score, 0.001) // max/2 each

	// Perfect: teórico groupScore 60 => 30+30*(60-24)/56 = 49.29 (NOT 60).
	// Práctico groupScore 40 => 20+20*(40-18)/22 = 40 (reaches max).
	full := merge(answerAll(questions, 80, 0, 1), answerAll(questions, 40, 0, 2))
	bd, err = CalculateGroupedXuntaBreakdown(groups, questions, full)
	assert.NoError(t, err)
	assert.InDelta(t, 49.29+40.0, bd.Score, 0.001)
}

func TestGroupedXunta_BelowCut(t *testing.T) {
	groups, questions := buildXuntaExam()
	// All blank: groupScore 0. Card 0; below cut => not passed; final follows the
	// line (30 + 30*(0-24)/56 = 17.142857), never the displayed 0.
	bd, err := CalculateGroupedXuntaBreakdown(groups, questions, map[uint]string{})
	assert.NoError(t, err)
	assert.InDelta(t, 0.0, bd.Groups[0].Score, 0.001)
	assert.False(t, bd.Groups[0].Passed)
	assert.InDelta(t, 17.14+3.64, bd.Score, 0.001) // 20.78
}

func TestGroupedBreakdown_BackCompatFlatMatchesUngrouped(t *testing.T) {
	// A flat exam scored via the legacy path must be untouched by grouping: the
	// grouped path is only entered when groups exist.
	questions := []models.Question{
		{ID: 1, IsActive: true, CorrectOption: "A"},
		{ID: 2, IsActive: true, CorrectOption: "B"},
		{ID: 3, IsActive: true, CorrectOption: "C"},
		{ID: 4, IsActive: true, CorrectOption: "D"},
	}
	answers := map[uint]string{1: "A", 2: "B", 3: "C", 4: "A"} // 3 correct, 1 wrong

	flat, err := CalculateScoreBreakdown(questions, answers, false, 0, 100)
	assert.NoError(t, err)
	assert.Equal(t, 75.0, flat.Score)
	assert.Nil(t, flat.Groups)
}
