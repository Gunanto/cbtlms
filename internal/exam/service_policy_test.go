package exam

import (
	"testing"
	"time"
)

func TestCanViewResultByPolicy(t *testing.T) {
	now := time.Now()
	past := now.Add(-time.Minute)
	future := now.Add(time.Minute)

	tests := []struct {
		name         string
		policy       string
		status       string
		examEndAt    *time.Time
		expectAllow  bool
	}{
		{name: "after_submit submitted", policy: "after_submit", status: "submitted", expectAllow: true},
		{name: "after_submit in_progress", policy: "after_submit", status: "in_progress", expectAllow: false},
		{name: "after_exam_end before_end", policy: "after_exam_end", status: "submitted", examEndAt: &future, expectAllow: false},
		{name: "after_exam_end after_end", policy: "after_exam_end", status: "submitted", examEndAt: &past, expectAllow: true},
		{name: "after_exam_end no_end", policy: "after_exam_end", status: "submitted", examEndAt: nil, expectAllow: false},
		{name: "immediate finalized", policy: "immediate", status: "expired", expectAllow: true},
		{name: "disabled finalized", policy: "disabled", status: "submitted", expectAllow: false},
		{name: "unknown policy fallback", policy: "other", status: "submitted", expectAllow: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := canViewResultByPolicy(tc.policy, tc.status, tc.examEndAt)
			if got != tc.expectAllow {
				t.Fatalf("expected %v, got %v", tc.expectAllow, got)
			}
		})
	}
}
