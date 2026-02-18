package question

import (
	"encoding/json"
	"testing"
)

func TestValidateStimulusContentSingle(t *testing.T) {
	err := validateStimulusContent("single", json.RawMessage(`{"body":"Isi stimulus"}`))
	if err != nil {
		t.Fatalf("expected valid single content, got err=%v", err)
	}
}

func TestValidateStimulusContentMultiteksInvalid(t *testing.T) {
	err := validateStimulusContent("multiteks", json.RawMessage(`{"tabs":[]}`))
	if err == nil {
		t.Fatalf("expected error for empty multiteks tabs")
	}
}

func TestValidateAnswerKey(t *testing.T) {
	tests := []struct {
		name         string
		questionType string
		answerKey    string
		wantErr      bool
	}{
		{name: "pg_tunggal valid", questionType: "pg_tunggal", answerKey: `{"correct":"B"}`, wantErr: false},
		{name: "pg_tunggal invalid", questionType: "pg_tunggal", answerKey: `{"correct":[]}`, wantErr: true},
		{name: "multi valid exact", questionType: "multi_jawaban", answerKey: `{"correct":["A","D"],"mode":"exact"}`, wantErr: false},
		{name: "multi valid without mode", questionType: "multi_jawaban", answerKey: `{"correct":["A","D"]}`, wantErr: false},
		{name: "multi invalid mode", questionType: "multi_jawaban", answerKey: `{"correct":["A","D"],"mode":"partial"}`, wantErr: true},
		{name: "multi invalid duplicate", questionType: "multi_jawaban", answerKey: `{"correct":["A","A"]}`, wantErr: true},
		{name: "bs valid", questionType: "benar_salah_pernyataan", answerKey: `{"statements":[{"id":"s1","correct":true},{"id":"s2","correct":false}]}`, wantErr: false},
		{name: "bs invalid duplicate id", questionType: "benar_salah_pernyataan", answerKey: `{"statements":[{"id":"s1","correct":true},{"id":"s1","correct":false}]}`, wantErr: true},
		{name: "bs invalid empty", questionType: "benar_salah_pernyataan", answerKey: `{"statements":[]}`, wantErr: true},
		{name: "unsupported type", questionType: "isian", answerKey: `{"correct":"x"}`, wantErr: true},
		{name: "invalid json", questionType: "pg_tunggal", answerKey: `{"correct":`, wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := validateAnswerKey(tc.questionType, json.RawMessage(tc.answerKey))
			if tc.wantErr && err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
		})
	}
}
