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

	bd, err := CalculateGroupedBreakdown(groups, questions, answers)
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
	// Teórico: ppc = 60/80 = 0.75. 50 correct, 30 wrong => 50*0.75 - 30*0.25 = 37.5 - 7.5 = 30 (justo el mínimo, pasa).
	// Bajamos a 48 correct, 32 wrong => 36 - 8 = 28 < 30 => falla eliminatorio.
	answers := merge(answerAll(questions, 48, 32, 1), answerAll(questions, 40, 0, 2))

	bd, err := CalculateGroupedBreakdown(groups, questions, answers)
	assert.NoError(t, err)
	assert.InDelta(t, 28.0, bd.Groups[0].Score, 0.001)
	assert.False(t, bd.Groups[0].Passed)
	assert.True(t, bd.Groups[1].Passed)
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
