package exam

import (
	"testing"

	"github.com/inscripcion-moodle/go-backend/internal/models"
	"github.com/stretchr/testify/assert"
)

func TestCalculateScoreBreakdown(t *testing.T) {
	questions := []models.Question{
		{ID: 1, IsActive: true, CorrectOption: "A"},
		{ID: 2, IsActive: true, CorrectOption: "B"},
		{ID: 3, IsActive: true, CorrectOption: "C"},
		{ID: 4, IsActive: true, CorrectOption: "D"},
	}

	tests := []struct {
		name      string
		answers   map[uint]string
		subtracts bool
		penalty   float64
		maxScore  float64
		wantScore float64
	}{
		{
			name: "No penalty, 3 correct, 1 wrong (Max 100)",
			answers: map[uint]string{
				1: "A", 2: "B", 3: "C", 4: "A", // 4 is wrong
			},
			subtracts: false,
			penalty:   0,
			maxScore:  100.0,
			wantScore: 75.0, // 3/4 * 100
		},
		{
			name: "Penalty 0.25, 3 correct, 1 wrong",
			answers: map[uint]string{
				1: "A", 2: "B", 3: "C", 4: "A", // 4 is wrong
			},
			subtracts: true,
			penalty:   0.25,
			maxScore:  100.0,
			wantScore: 68.75, // (3 - 0.25) / 4 * 100 = 2.75 / 4 * 100
		},
		{
			name: "Penalty 0.5, 2 correct, 2 wrong",
			answers: map[uint]string{
				1: "A", 2: "B", 3: "A", 4: "A", // 3, 4 wrong
			},
			subtracts: true,
			penalty:   0.5,
			maxScore:  100.0,
			wantScore: 25.0, // (2 - 1) / 4 * 100 = 1 / 4 * 100
		},
		{
			name: "Penalty 0.5, 2 correct, 1 wrong, 1 blank",
			answers: map[uint]string{
				1: "A", 2: "B", 3: "A", // 3 wrong, 4 blank
			},
			subtracts: true,
			penalty:   0.5,
			maxScore:  100.0,
			wantScore: 37.5, // (2 - 0.5) / 4 * 100 = 1.5 / 4 * 100
		},
		{
			name: "Penalty 1.0, All wrong",
			answers: map[uint]string{
				1: "B", 2: "A", 3: "A", 4: "A", // All wrong
			},
			subtracts: true,
			penalty:   1.0,
			maxScore:  100.0,
			wantScore: -100.0, // (0 - 4) / 4 * 100
		},
		{
			name: "MaxScore 10, 3 correct, 1 wrong",
			answers: map[uint]string{
				1: "A", 2: "B", 3: "C", 4: "A", // 4 is wrong
			},
			subtracts: false,
			penalty:   0,
			maxScore:  10.0,
			wantScore: 7.5, // 3/4 * 10 = 7.5
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CalculateScoreBreakdown(questions, tt.answers, tt.subtracts, tt.penalty, tt.maxScore)
			assert.NoError(t, err)
			assert.Equal(t, tt.wantScore, got.Score)
		})
	}
}
