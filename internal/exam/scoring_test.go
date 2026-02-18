package exam

import "testing"

func TestScoreQuestion_PGTunggal(t *testing.T) {
	tests := []struct {
		name      string
		key       string
		payload   string
		fallback  []string
		weight    float64
		reason    string
		answered  bool
		earned    float64
		isCorrect *bool
	}{
		{name: "correct string payload", key: `{"correct":"B"}`, payload: `{"selected":"B"}`, weight: 2, reason: "correct", answered: true, earned: 2, isCorrect: boolPtr(true)},
		{name: "wrong string payload", key: `{"correct":"B"}`, payload: `{"selected":"A"}`, weight: 2, reason: "wrong", answered: true, earned: 0, isCorrect: boolPtr(false)},
		{name: "correct single array", key: `{"correct":"B"}`, payload: `{"selected":["B"]}`, weight: 1.5, reason: "correct", answered: true, earned: 1.5, isCorrect: boolPtr(true)},
		{name: "malformed multi array", key: `{"correct":"B"}`, payload: `{"selected":["A","B"]}`, weight: 1, reason: "malformed_payload", answered: true, earned: 0, isCorrect: boolPtr(false)},
		{name: "unanswered empty", key: `{"correct":"B"}`, payload: `{"selected":""}`, weight: 1, reason: "unanswered", answered: false, earned: 0, isCorrect: nil},
		{name: "unanswered missing selected", key: `{"correct":"B"}`, payload: `{}`, weight: 1, reason: "unanswered", answered: false, earned: 0, isCorrect: nil},
		{name: "malformed invalid json", key: `{"correct":"B"}`, payload: `{"selected":`, weight: 1, reason: "malformed_payload", answered: true, earned: 0, isCorrect: boolPtr(false)},
		{name: "fallback correct keys", key: `{}`, payload: `{"selected":"C"}`, fallback: []string{"C"}, weight: 3, reason: "correct", answered: true, earned: 3, isCorrect: boolPtr(true)},
		{name: "malformed answer key", key: `{}`, payload: `{"selected":"A"}`, fallback: nil, weight: 1, reason: "malformed_answer_key", answered: false, earned: 0, isCorrect: nil},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := ScoreQuestion(ScoreInput{
				QuestionType:  "pg_tunggal",
				AnswerKey:     []byte(tc.key),
				AnswerPayload: []byte(tc.payload),
				CorrectKeys:   tc.fallback,
				Weight:        tc.weight,
			})
			assertScoreResult(t, got, tc.reason, tc.answered, tc.earned, tc.isCorrect)
		})
	}
}

func TestScoreQuestion_MultiJawabanExact(t *testing.T) {
	tests := []struct {
		name      string
		key       string
		payload   string
		fallback  []string
		weight    float64
		reason    string
		answered  bool
		earned    float64
		isCorrect *bool
	}{
		{name: "exact correct", key: `{"correct":["A","D"],"mode":"exact"}`, payload: `{"selected":["D","A"]}`, weight: 4, reason: "correct", answered: true, earned: 4, isCorrect: boolPtr(true)},
		{name: "wrong missing one", key: `{"correct":["A","D"],"mode":"exact"}`, payload: `{"selected":["A"]}`, weight: 4, reason: "wrong", answered: true, earned: 0, isCorrect: boolPtr(false)},
		{name: "wrong extra one", key: `{"correct":["A","D"],"mode":"exact"}`, payload: `{"selected":["A","D","B"]}`, weight: 4, reason: "wrong", answered: true, earned: 0, isCorrect: boolPtr(false)},
		{name: "unanswered empty list", key: `{"correct":["A","D"],"mode":"exact"}`, payload: `{"selected":[]}`, weight: 4, reason: "unanswered", answered: false, earned: 0, isCorrect: nil},
		{name: "unanswered missing selected", key: `{"correct":["A","D"],"mode":"exact"}`, payload: `{}`, weight: 4, reason: "unanswered", answered: false, earned: 0, isCorrect: nil},
		{name: "malformed payload non-array", key: `{"correct":["A","D"],"mode":"exact"}`, payload: `{"selected":123}`, weight: 4, reason: "unanswered", answered: false, earned: 0, isCorrect: nil},
		{name: "fallback key works", key: `{}`, payload: `{"selected":["C"]}`, fallback: []string{"C"}, weight: 2, reason: "correct", answered: true, earned: 2, isCorrect: boolPtr(true)},
		{name: "unknown mode forced exact", key: `{"correct":["A","D"],"mode":"partial"}`, payload: `{"selected":["A"]}`, weight: 4, reason: "wrong", answered: true, earned: 0, isCorrect: boolPtr(false)},
		{name: "malformed answer key", key: `{"correct":[]}`, payload: `{"selected":["A"]}`, weight: 1, reason: "malformed_answer_key", answered: false, earned: 0, isCorrect: nil},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := ScoreQuestion(ScoreInput{
				QuestionType:  "multi_jawaban",
				AnswerKey:     []byte(tc.key),
				AnswerPayload: []byte(tc.payload),
				CorrectKeys:   tc.fallback,
				Weight:        tc.weight,
			})
			assertScoreResult(t, got, tc.reason, tc.answered, tc.earned, tc.isCorrect)
		})
	}
}

func TestScoreQuestion_BenarSalahPernyataan(t *testing.T) {
	tests := []struct {
		name           string
		key            string
		payload        string
		weight         float64
		reason         string
		answered       bool
		earned         float64
		isCorrect      *bool
		expectBreakLen int
	}{
		{
			name:    "all correct",
			key:     `{"statements":[{"id":"s1","correct":true},{"id":"s2","correct":false}]}`,
			payload: `{"answers":[{"id":"s1","value":true},{"id":"s2","value":false}]}`,
			weight:  2,
			reason:  "correct", answered: true, earned: 2, isCorrect: boolPtr(true), expectBreakLen: 2,
		},
		{
			name:    "partial",
			key:     `{"statements":[{"id":"s1","correct":true},{"id":"s2","correct":false}]}`,
			payload: `{"answers":[{"id":"s1","value":true},{"id":"s2","value":true}]}`,
			weight:  2,
			reason:  "partial", answered: true, earned: 1, isCorrect: boolPtr(false), expectBreakLen: 2,
		},
		{
			name:    "wrong all",
			key:     `{"statements":[{"id":"s1","correct":true},{"id":"s2","correct":false}]}`,
			payload: `{"answers":[{"id":"s1","value":false},{"id":"s2","value":true}]}`,
			weight:  2,
			reason:  "wrong", answered: true, earned: 0, isCorrect: boolPtr(false), expectBreakLen: 2,
		},
		{
			name:    "unanswered missing answers",
			key:     `{"statements":[{"id":"s1","correct":true}]}`,
			payload: `{}`,
			weight:  1,
			reason:  "unanswered", answered: false, earned: 0, isCorrect: nil,
		},
		{
			name:    "malformed payload missing statement",
			key:     `{"statements":[{"id":"s1","correct":true},{"id":"s2","correct":false}]}`,
			payload: `{"answers":[{"id":"s1","value":true}]}`,
			weight:  2,
			reason:  "malformed_payload", answered: true, earned: 0, isCorrect: boolPtr(false),
		},
		{
			name:    "malformed payload duplicate ids",
			key:     `{"statements":[{"id":"s1","correct":true}]}`,
			payload: `{"answers":[{"id":"s1","value":true},{"id":"s1","value":false}]}`,
			weight:  1,
			reason:  "malformed_payload", answered: true, earned: 0, isCorrect: boolPtr(false),
		},
		{
			name:    "malformed key",
			key:     `{"statements":[{"id":"","correct":true}]}`,
			payload: `{"answers":[{"id":"s1","value":true}]}`,
			weight:  1,
			reason:  "malformed_answer_key", answered: false, earned: 0, isCorrect: nil,
		},
		{
			name:    "zero weight still deterministic",
			key:     `{"statements":[{"id":"s1","correct":true},{"id":"s2","correct":false}]}`,
			payload: `{"answers":[{"id":"s1","value":true},{"id":"s2","value":false}]}`,
			weight:  0,
			reason:  "correct", answered: true, earned: 0, isCorrect: boolPtr(true), expectBreakLen: 2,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := ScoreQuestion(ScoreInput{
				QuestionType:  "benar_salah_pernyataan",
				AnswerKey:     []byte(tc.key),
				AnswerPayload: []byte(tc.payload),
				Weight:        tc.weight,
			})
			assertScoreResult(t, got, tc.reason, tc.answered, tc.earned, tc.isCorrect)
			if tc.expectBreakLen > 0 && len(got.Breakdown) != tc.expectBreakLen {
				t.Fatalf("expected breakdown len %d, got %d", tc.expectBreakLen, len(got.Breakdown))
			}
		})
	}
}

func assertScoreResult(t *testing.T, got ScoreResult, reason string, answered bool, earned float64, isCorrect *bool) {
	t.Helper()
	if got.Reason != reason {
		t.Fatalf("expected reason=%s, got=%s", reason, got.Reason)
	}
	if got.Answered != answered {
		t.Fatalf("expected answered=%v, got=%v", answered, got.Answered)
	}
	if got.EarnedScore != earned {
		t.Fatalf("expected earned=%v, got=%v", earned, got.EarnedScore)
	}
	if isCorrect == nil {
		if got.IsCorrect != nil {
			t.Fatalf("expected is_correct=nil, got=%v", *got.IsCorrect)
		}
		return
	}
	if got.IsCorrect == nil {
		t.Fatalf("expected is_correct=%v, got=nil", *isCorrect)
	}
	if *got.IsCorrect != *isCorrect {
		t.Fatalf("expected is_correct=%v, got=%v", *isCorrect, *got.IsCorrect)
	}
}
