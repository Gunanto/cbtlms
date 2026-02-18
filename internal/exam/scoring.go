package exam

// ScoringResult contains minimum fields for objective scoring.
type ScoringResult struct {
	Correct int
	Wrong   int
	Score   float64
}

func ScoreObjective(correct, wrong int, weight float64) ScoringResult {
	if weight <= 0 {
		weight = 1
	}
	return ScoringResult{
		Correct: correct,
		Wrong:   wrong,
		Score:   float64(correct) * weight,
	}
}
