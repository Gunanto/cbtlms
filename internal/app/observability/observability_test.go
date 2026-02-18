package observability

import "testing"

func TestNormalizedPath(t *testing.T) {
	got := normalizedPath("/api/v1/attempts/123/answers/9")
	want := "/api/v1/attempts/{id}/answers/{id}"
	if got != want {
		t.Fatalf("normalizedPath mismatch got=%s want=%s", got, want)
	}
}

func TestExtractAttemptID(t *testing.T) {
	if id := extractAttemptID("/api/v1/attempts/456/submit"); id != 456 {
		t.Fatalf("expected 456, got %d", id)
	}
	if id := extractAttemptID("/api/v1/exams/1"); id != 0 {
		t.Fatalf("expected 0 for non-attempt path, got %d", id)
	}
}
