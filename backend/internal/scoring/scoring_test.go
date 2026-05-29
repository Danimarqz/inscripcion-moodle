package scoring

import "testing"

func TestComputeScore(t *testing.T) {
	tests := []struct {
		name                      string
		correct, incorrect, total int
		cfg                       Config
		want                      float64
	}{
		// --- legacy parity (mirrors services/exam/calculation_test.go) ---
		{"legacy 3/1 no penalty", 3, 1, 4, Config{Mode: "legacy", MaxScore: 100}, 75.0},
		{"legacy all wrong penalty 1.0 -> -100 (no floor)", 0, 4, 4, Config{Mode: "legacy", Subtracts: true, Penalty: 1.0, MaxScore: 100}, -100.0},
		{"legacy max 10", 3, 1, 4, Config{Mode: "legacy", MaxScore: 10}, 7.5},
		{"empty mode treated as legacy", 3, 1, 4, Config{Mode: "", MaxScore: 100}, 75.0},
		{"total 0 -> 0 (legacy)", 0, 0, 0, Config{Mode: "legacy", MaxScore: 100}, 0},

		// --- absolute ---
		{"absolute 3 correct 1 wrong", 3, 1, 4, Config{Mode: "absolute", PointsPerCorrect: 0.40, PointsPerWrong: 0.10}, 1.10},
		{"absolute all correct", 4, 0, 4, Config{Mode: "absolute", PointsPerCorrect: 0.40, PointsPerWrong: 0.10}, 1.60},
		{"absolute floors at 0", 0, 4, 4, Config{Mode: "absolute", PointsPerCorrect: 0.40, PointsPerWrong: 0.10}, 0},
		{"absolute ppw 0 = no deduction", 3, 1, 4, Config{Mode: "absolute", PointsPerCorrect: 0.40, PointsPerWrong: 0}, 1.20},
		{"absolute rounds to 2dp", 1, 0, 3, Config{Mode: "absolute", PointsPerCorrect: 0.333333, PointsPerWrong: 0}, 0.33},
		{"absolute total 0 -> 0", 0, 0, 0, Config{Mode: "absolute", PointsPerCorrect: 0.40, PointsPerWrong: 0.10}, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ComputeScore(tt.correct, tt.incorrect, tt.total, tt.cfg)
			if got != tt.want {
				t.Fatalf("ComputeScore() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConfigFromExamFields(t *testing.T) {
	f := func(v float64) *float64 { return &v }

	// nil maxScore -> default 100; empty mode -> legacy.
	cfg := ConfigFromExamFields("", true, f(0.25), nil, nil, nil)
	if cfg.Mode != "legacy" {
		t.Fatalf("mode = %q, want legacy", cfg.Mode)
	}
	if cfg.MaxScore != 100 {
		t.Fatalf("maxScore = %v, want 100", cfg.MaxScore)
	}
	if cfg.Penalty != 0.25 {
		t.Fatalf("penalty = %v, want 0.25", cfg.Penalty)
	}

	// absolute fields resolved.
	cfg = ConfigFromExamFields("absolute", false, nil, nil, f(0.40), f(0.10))
	if cfg.Mode != "absolute" || cfg.PointsPerCorrect != 0.40 || cfg.PointsPerWrong != 0.10 {
		t.Fatalf("unexpected absolute cfg: %+v", cfg)
	}
}
