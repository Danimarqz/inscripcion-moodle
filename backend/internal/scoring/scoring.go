// Package scoring is the single source of truth for exam score computation.
// It is a dependency-free leaf package so both the repository and the exam
// service can use it without an import cycle.
package scoring

import "math"

// Config holds the resolved scoring parameters for one exam.
type Config struct {
	Mode             string  // "" | "legacy" | "absolute"
	Subtracts        bool    // legacy: whether wrong answers subtract
	Penalty          float64 // legacy: fraction of one correct answer
	MaxScore         float64 // legacy: max score (default resolved to 100)
	PointsPerCorrect float64 // absolute: points added per correct answer
	PointsPerWrong   float64 // absolute: points subtracted per wrong answer (positive magnitude)
}

// ConfigFromExamFields resolves exam fields (with nil pointers) into a Config,
// applying defaults: empty mode -> "legacy", MaxScore default 100. Centralizes
// the nil-resolution so every call site behaves identically.
func ConfigFromExamFields(mode string, subtracts bool, penalty, maxScore, pointsPerCorrect, pointsPerWrong *float64) Config {
	cfg := Config{Mode: mode, Subtracts: subtracts, MaxScore: 100.0}
	if cfg.Mode == "" {
		cfg.Mode = "legacy"
	}
	if penalty != nil {
		cfg.Penalty = *penalty
	}
	if maxScore != nil {
		cfg.MaxScore = *maxScore
	}
	if pointsPerCorrect != nil {
		cfg.PointsPerCorrect = *pointsPerCorrect
	}
	if pointsPerWrong != nil {
		cfg.PointsPerWrong = *pointsPerWrong
	}
	return cfg
}

// ComputeScore returns the stored score for one submission, rounded to 2 dp.
//
//   - legacy:   (correct - penalty*wrong) / total * maxScore. Can go negative
//     (the historical contract: all-wrong on penalty 1.0 yields -maxScore).
//   - absolute: correct*pointsPerCorrect - wrong*pointsPerWrong, FLOORED at 0
//     (product decision: an absolute score never drops below 0).
//
// total == 0 yields 0 in both modes.
//
// NOTE: the legacy *breakdown* path in the exam service stays UNROUNDED to
// preserve exact create-time parity; this helper is the rounded variant used by
// the recalc paths and by the absolute breakdown branch.
func ComputeScore(correct, incorrect, total int, cfg Config) float64 {
	if total == 0 {
		return 0
	}
	var raw float64
	switch cfg.Mode {
	case "absolute":
		raw = float64(correct)*cfg.PointsPerCorrect - float64(incorrect)*cfg.PointsPerWrong
		if raw < 0 {
			raw = 0
		}
	default: // "" or "legacy"
		net := float64(correct)
		if cfg.Subtracts && cfg.Penalty > 0 {
			net -= float64(incorrect) * cfg.Penalty
		}
		raw = net / float64(total) * cfg.MaxScore
	}
	return math.Round(raw*100) / 100
}
