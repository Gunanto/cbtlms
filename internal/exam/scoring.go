package exam

import (
	"encoding/json"
	"sort"
	"strings"
)

type ScoreInput struct {
	QuestionType  string
	AnswerKey     []byte
	AnswerPayload []byte
	CorrectKeys   []string
	Weight        float64
}

type StatementScore struct {
	ID      string `json:"id"`
	Correct bool   `json:"correct"`
	Answer  *bool  `json:"answer,omitempty"`
}

type ScoreResult struct {
	Answered    bool             `json:"answered"`
	IsCorrect   *bool            `json:"is_correct,omitempty"`
	EarnedScore float64          `json:"earned_score"`
	Reason      string           `json:"reason"`
	Selected    []string         `json:"selected,omitempty"`
	Correct     []string         `json:"correct,omitempty"`
	Breakdown   []StatementScore `json:"breakdown,omitempty"`
}

func ScoreQuestion(in ScoreInput) ScoreResult {
	qType := strings.TrimSpace(strings.ToLower(in.QuestionType))
	weight := in.Weight
	if weight < 0 {
		weight = 0
	}

	switch qType {
	case "pg_tunggal":
		return scorePGTunggal(in.AnswerKey, in.AnswerPayload, in.CorrectKeys, weight)
	case "multi_jawaban":
		return scoreMultiJawaban(in.AnswerKey, in.AnswerPayload, in.CorrectKeys, weight)
	case "benar_salah_pernyataan":
		return scoreBenarSalahPernyataan(in.AnswerKey, in.AnswerPayload, weight)
	default:
		return scorePGTunggal(in.AnswerKey, in.AnswerPayload, in.CorrectKeys, weight)
	}
}

func scorePGTunggal(answerKeyRaw, payloadRaw []byte, fallbackCorrect []string, weight float64) ScoreResult {
	correct, okKey := parsePGTunggalKey(answerKeyRaw)
	if !okKey {
		correct = firstClean(fallbackCorrect)
	}
	if correct == "" {
		return ScoreResult{Reason: "malformed_answer_key"}
	}

	selected, status := parseSingleSelection(payloadRaw)
	if status == "unanswered" {
		return ScoreResult{Answered: false, IsCorrect: nil, EarnedScore: 0, Reason: "unanswered", Correct: []string{correct}}
	}
	if status == "malformed" {
		f := false
		return ScoreResult{Answered: true, IsCorrect: &f, EarnedScore: 0, Reason: "malformed_payload", Correct: []string{correct}}
	}

	isCorrect := strings.EqualFold(selected, correct)
	if isCorrect {
		return ScoreResult{Answered: true, IsCorrect: boolPtr(true), EarnedScore: weight, Reason: "correct", Selected: []string{selected}, Correct: []string{correct}}
	}
	return ScoreResult{Answered: true, IsCorrect: boolPtr(false), EarnedScore: 0, Reason: "wrong", Selected: []string{selected}, Correct: []string{correct}}
}

func scoreMultiJawaban(answerKeyRaw, payloadRaw []byte, fallbackCorrect []string, weight float64) ScoreResult {
	correctSet, mode, okKey := parseMultiJawabanKey(answerKeyRaw)
	if !okKey {
		correctSet = normalizeStringSet(fallbackCorrect)
		mode = "exact"
	}
	if len(correctSet) == 0 {
		return ScoreResult{Reason: "malformed_answer_key"}
	}
	if mode == "" {
		mode = "exact"
	}

	selectedSet, status := parseMultiSelection(payloadRaw)
	if status == "unanswered" {
		return ScoreResult{Answered: false, IsCorrect: nil, EarnedScore: 0, Reason: "unanswered", Correct: correctSet}
	}
	if status == "malformed" {
		f := false
		return ScoreResult{Answered: true, IsCorrect: &f, EarnedScore: 0, Reason: "malformed_payload", Correct: correctSet}
	}

	if mode != "exact" {
		mode = "exact"
	}
	isCorrect := equalSet(selectedSet, correctSet)
	if isCorrect {
		return ScoreResult{Answered: true, IsCorrect: boolPtr(true), EarnedScore: weight, Reason: "correct", Selected: selectedSet, Correct: correctSet}
	}
	return ScoreResult{Answered: true, IsCorrect: boolPtr(false), EarnedScore: 0, Reason: "wrong", Selected: selectedSet, Correct: correctSet}
}

func scoreBenarSalahPernyataan(answerKeyRaw, payloadRaw []byte, weight float64) ScoreResult {
	keyMap, keyOrder, okKey := parseBenarSalahKey(answerKeyRaw)
	if !okKey || len(keyOrder) == 0 {
		return ScoreResult{Reason: "malformed_answer_key"}
	}

	answerMap, status := parseBenarSalahAnswers(payloadRaw)
	if status == "unanswered" {
		return ScoreResult{Answered: false, IsCorrect: nil, EarnedScore: 0, Reason: "unanswered", Correct: formatStatementPairs(keyMap, keyOrder)}
	}
	if status == "malformed" {
		f := false
		return ScoreResult{Answered: true, IsCorrect: &f, EarnedScore: 0, Reason: "malformed_payload", Correct: formatStatementPairs(keyMap, keyOrder)}
	}

	for _, id := range keyOrder {
		if _, ok := answerMap[id]; !ok {
			f := false
			return ScoreResult{Answered: true, IsCorrect: &f, EarnedScore: 0, Reason: "malformed_payload", Correct: formatStatementPairs(keyMap, keyOrder)}
		}
	}

	perStatement := 0.0
	if len(keyOrder) > 0 {
		perStatement = weight / float64(len(keyOrder))
	}

	correctCount := 0
	breakdown := make([]StatementScore, 0, len(keyOrder))
	selected := make([]string, 0, len(keyOrder))
	for _, id := range keyOrder {
		answer := answerMap[id]
		expected := keyMap[id]
		if answer == expected {
			correctCount++
		}
		selected = append(selected, statementPair(id, answer))
		val := answer
		breakdown = append(breakdown, StatementScore{ID: id, Correct: answer == expected, Answer: &val})
	}

	earned := perStatement * float64(correctCount)
	allCorrect := correctCount == len(keyOrder)
	reason := "partial"
	if allCorrect {
		reason = "correct"
	} else if correctCount == 0 {
		reason = "wrong"
	}

	return ScoreResult{
		Answered:    true,
		IsCorrect:   boolPtr(allCorrect),
		EarnedScore: earned,
		Reason:      reason,
		Selected:    selected,
		Correct:     formatStatementPairs(keyMap, keyOrder),
		Breakdown:   breakdown,
	}
}

func parsePGTunggalKey(raw []byte) (string, bool) {
	if len(raw) == 0 {
		return "", false
	}
	var obj map[string]interface{}
	if err := json.Unmarshal(raw, &obj); err != nil {
		return "", false
	}
	v, ok := obj["correct"]
	if !ok {
		return "", false
	}
	s, ok := v.(string)
	if !ok {
		return "", false
	}
	s = strings.TrimSpace(s)
	if s == "" {
		return "", false
	}
	return s, true
}

func parseMultiJawabanKey(raw []byte) ([]string, string, bool) {
	if len(raw) == 0 {
		return nil, "", false
	}
	var obj map[string]interface{}
	if err := json.Unmarshal(raw, &obj); err != nil {
		return nil, "", false
	}
	mode := "exact"
	if v, ok := obj["mode"]; ok {
		if s, ok := v.(string); ok {
			mode = strings.TrimSpace(strings.ToLower(s))
		}
	}
	v, ok := obj["correct"]
	if !ok {
		return nil, mode, false
	}
	list := anyToStringSlice(v)
	if len(list) == 0 {
		return nil, mode, false
	}
	return list, mode, true
}

func parseBenarSalahKey(raw []byte) (map[string]bool, []string, bool) {
	if len(raw) == 0 {
		return nil, nil, false
	}
	var obj map[string]interface{}
	if err := json.Unmarshal(raw, &obj); err != nil {
		return nil, nil, false
	}
	v, ok := obj["statements"]
	if !ok {
		return nil, nil, false
	}
	arr, ok := v.([]interface{})
	if !ok || len(arr) == 0 {
		return nil, nil, false
	}

	out := make(map[string]bool, len(arr))
	order := make([]string, 0, len(arr))
	for _, it := range arr {
		item, ok := it.(map[string]interface{})
		if !ok {
			return nil, nil, false
		}
		id, _ := item["id"].(string)
		id = strings.TrimSpace(id)
		val, ok := item["correct"].(bool)
		if id == "" || !ok {
			return nil, nil, false
		}
		if _, exists := out[id]; exists {
			return nil, nil, false
		}
		out[id] = val
		order = append(order, id)
	}

	return out, order, true
}

func parseSingleSelection(raw []byte) (string, string) {
	if len(raw) == 0 {
		return "", "unanswered"
	}
	var obj map[string]interface{}
	if err := json.Unmarshal(raw, &obj); err != nil {
		return "", "malformed"
	}
	v, ok := obj["selected"]
	if !ok {
		return "", "unanswered"
	}
	switch t := v.(type) {
	case string:
		t = strings.TrimSpace(t)
		if t == "" {
			return "", "unanswered"
		}
		return t, "answered"
	case []interface{}:
		if len(t) == 0 {
			return "", "unanswered"
		}
		if len(t) > 1 {
			return "", "malformed"
		}
		s, ok := t[0].(string)
		if !ok {
			return "", "malformed"
		}
		s = strings.TrimSpace(s)
		if s == "" {
			return "", "unanswered"
		}
		return s, "answered"
	default:
		return "", "malformed"
	}
}

func parseMultiSelection(raw []byte) ([]string, string) {
	if len(raw) == 0 {
		return nil, "unanswered"
	}
	var obj map[string]interface{}
	if err := json.Unmarshal(raw, &obj); err != nil {
		return nil, "malformed"
	}
	v, ok := obj["selected"]
	if !ok {
		return nil, "unanswered"
	}
	list := anyToStringSlice(v)
	if len(list) == 0 {
		return nil, "unanswered"
	}
	return list, "answered"
}

func parseBenarSalahAnswers(raw []byte) (map[string]bool, string) {
	if len(raw) == 0 {
		return nil, "unanswered"
	}
	var obj map[string]interface{}
	if err := json.Unmarshal(raw, &obj); err != nil {
		return nil, "malformed"
	}
	v, ok := obj["answers"]
	if !ok {
		return nil, "unanswered"
	}
	arr, ok := v.([]interface{})
	if !ok {
		return nil, "malformed"
	}
	if len(arr) == 0 {
		return nil, "unanswered"
	}

	out := make(map[string]bool, len(arr))
	for _, it := range arr {
		item, ok := it.(map[string]interface{})
		if !ok {
			return nil, "malformed"
		}
		id, _ := item["id"].(string)
		id = strings.TrimSpace(id)
		val, ok := item["value"].(bool)
		if id == "" || !ok {
			return nil, "malformed"
		}
		if _, exists := out[id]; exists {
			return nil, "malformed"
		}
		out[id] = val
	}
	return out, "answered"
}

func normalizeStringSet(in []string) []string {
	set := map[string]struct{}{}
	for _, v := range in {
		s := strings.TrimSpace(v)
		if s == "" {
			continue
		}
		set[s] = struct{}{}
	}
	out := make([]string, 0, len(set))
	for k := range set {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func firstClean(in []string) string {
	for _, s := range in {
		s = strings.TrimSpace(s)
		if s != "" {
			return s
		}
	}
	return ""
}

func formatStatementPairs(keyMap map[string]bool, order []string) []string {
	out := make([]string, 0, len(order))
	for _, id := range order {
		out = append(out, statementPair(id, keyMap[id]))
	}
	return out
}

func statementPair(id string, value bool) string {
	if value {
		return id + ":true"
	}
	return id + ":false"
}

func boolPtr(v bool) *bool {
	return &v
}
