package exam

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"cbtlms/internal/auth"

	"github.com/go-chi/chi/v5"
)

type mockExamService struct {
	startAttemptFn                  func(ctx context.Context, examID, studentID int64, examToken string) (*Attempt, error)
	listAdminExamsFn                func(ctx context.Context, includeInactive bool) ([]ExamAdminRecord, error)
	createExamFn                    func(ctx context.Context, in CreateExamInput) (*ExamAdminRecord, error)
	updateExamFn                    func(ctx context.Context, in UpdateExamInput) (*ExamAdminRecord, error)
	deleteExamFn                    func(ctx context.Context, examID int64) error
	listSubjectsFn                  func(ctx context.Context, level, subjectType string) ([]SubjectOption, error)
	createSubjectFn                 func(ctx context.Context, in CreateSubjectInput) (*SubjectOption, error)
	updateSubjectFn                 func(ctx context.Context, in UpdateSubjectInput) (*SubjectOption, error)
	deleteSubjectFn                 func(ctx context.Context, subjectID int64) error
	listExamsFn                     func(ctx context.Context, subjectID int64) ([]ExamOption, error)
	listExamsForTokenFn             func(ctx context.Context) ([]ExamTokenExam, error)
	generateExamTokenFn             func(ctx context.Context, examID, generatedBy int64, ttlMinutes int) (*ExamAccessToken, error)
	listExamAssignmentsFn           func(ctx context.Context, examID int64) ([]ExamAssignmentUser, error)
	replaceExamAssignmentsFn        func(ctx context.Context, in ReplaceExamAssignmentsInput) ([]ExamAssignmentUser, error)
	replaceExamAssignmentsByClassFn func(ctx context.Context, in ReplaceExamAssignmentsByClassInput) ([]ExamAssignmentUser, error)
	listExamQuestionsFn             func(ctx context.Context, examID int64) ([]ExamQuestionManageItem, error)
	upsertExamQuestionFn            func(ctx context.Context, in UpsertExamQuestionInput) (*ExamQuestionManageItem, error)
	deleteExamQuestionFn            func(ctx context.Context, examID, questionID int64) error
	getAttemptSummaryFn             func(ctx context.Context, attemptID int64) (*AttemptSummary, error)
	getAttemptQuestionFn            func(ctx context.Context, attemptID int64, questionNo int) (*AttemptQuestion, error)
	getAttemptResultFn              func(ctx context.Context, attemptID int64) (*AttemptResult, error)
	saveAnswerFn                    func(ctx context.Context, input SaveAnswerInput) error
	submitAttemptFn                 func(ctx context.Context, attemptID int64) (*AttemptSummary, error)
	getAttemptOwnerFn               func(ctx context.Context, attemptID int64) (int64, error)
	logAttemptEventFn               func(ctx context.Context, input AttemptEventInput) (*AttemptEvent, error)
	listAttemptEventsFn             func(ctx context.Context, attemptID int64, limit int) ([]AttemptEvent, error)
}

func (m *mockExamService) StartAttempt(ctx context.Context, examID, studentID int64, examToken string) (*Attempt, error) {
	if m.startAttemptFn == nil {
		return nil, errors.New("not implemented")
	}
	return m.startAttemptFn(ctx, examID, studentID, examToken)
}

func (m *mockExamService) ListAdminExams(ctx context.Context, includeInactive bool) ([]ExamAdminRecord, error) {
	if m.listAdminExamsFn == nil {
		return nil, errors.New("not implemented")
	}
	return m.listAdminExamsFn(ctx, includeInactive)
}

func (m *mockExamService) CreateExam(ctx context.Context, in CreateExamInput) (*ExamAdminRecord, error) {
	if m.createExamFn == nil {
		return nil, errors.New("not implemented")
	}
	return m.createExamFn(ctx, in)
}

func (m *mockExamService) UpdateExam(ctx context.Context, in UpdateExamInput) (*ExamAdminRecord, error) {
	if m.updateExamFn == nil {
		return nil, errors.New("not implemented")
	}
	return m.updateExamFn(ctx, in)
}

func (m *mockExamService) DeleteExam(ctx context.Context, examID int64) error {
	if m.deleteExamFn == nil {
		return errors.New("not implemented")
	}
	return m.deleteExamFn(ctx, examID)
}

func (m *mockExamService) ListSubjects(ctx context.Context, level, subjectType string) ([]SubjectOption, error) {
	if m.listSubjectsFn == nil {
		return nil, errors.New("not implemented")
	}
	return m.listSubjectsFn(ctx, level, subjectType)
}

func (m *mockExamService) CreateSubject(ctx context.Context, in CreateSubjectInput) (*SubjectOption, error) {
	if m.createSubjectFn == nil {
		return nil, errors.New("not implemented")
	}
	return m.createSubjectFn(ctx, in)
}

func (m *mockExamService) UpdateSubject(ctx context.Context, in UpdateSubjectInput) (*SubjectOption, error) {
	if m.updateSubjectFn == nil {
		return nil, errors.New("not implemented")
	}
	return m.updateSubjectFn(ctx, in)
}

func (m *mockExamService) DeleteSubject(ctx context.Context, subjectID int64) error {
	if m.deleteSubjectFn == nil {
		return errors.New("not implemented")
	}
	return m.deleteSubjectFn(ctx, subjectID)
}

func (m *mockExamService) ListExamsBySubject(ctx context.Context, subjectID int64) ([]ExamOption, error) {
	if m.listExamsFn == nil {
		return nil, errors.New("not implemented")
	}
	return m.listExamsFn(ctx, subjectID)
}

func (m *mockExamService) ListExamsForToken(ctx context.Context) ([]ExamTokenExam, error) {
	if m.listExamsForTokenFn == nil {
		return nil, errors.New("not implemented")
	}
	return m.listExamsForTokenFn(ctx)
}

func (m *mockExamService) GenerateExamToken(ctx context.Context, examID, generatedBy int64, ttlMinutes int) (*ExamAccessToken, error) {
	if m.generateExamTokenFn == nil {
		return nil, errors.New("not implemented")
	}
	return m.generateExamTokenFn(ctx, examID, generatedBy, ttlMinutes)
}

func (m *mockExamService) ListExamAssignments(ctx context.Context, examID int64) ([]ExamAssignmentUser, error) {
	if m.listExamAssignmentsFn == nil {
		return nil, errors.New("not implemented")
	}
	return m.listExamAssignmentsFn(ctx, examID)
}

func (m *mockExamService) ReplaceExamAssignments(ctx context.Context, in ReplaceExamAssignmentsInput) ([]ExamAssignmentUser, error) {
	if m.replaceExamAssignmentsFn == nil {
		return nil, errors.New("not implemented")
	}
	return m.replaceExamAssignmentsFn(ctx, in)
}

func (m *mockExamService) ReplaceExamAssignmentsByClass(ctx context.Context, in ReplaceExamAssignmentsByClassInput) ([]ExamAssignmentUser, error) {
	if m.replaceExamAssignmentsByClassFn == nil {
		return nil, errors.New("not implemented")
	}
	return m.replaceExamAssignmentsByClassFn(ctx, in)
}

func (m *mockExamService) ListExamQuestions(ctx context.Context, examID int64) ([]ExamQuestionManageItem, error) {
	if m.listExamQuestionsFn == nil {
		return nil, errors.New("not implemented")
	}
	return m.listExamQuestionsFn(ctx, examID)
}

func (m *mockExamService) UpsertExamQuestion(ctx context.Context, in UpsertExamQuestionInput) (*ExamQuestionManageItem, error) {
	if m.upsertExamQuestionFn == nil {
		return nil, errors.New("not implemented")
	}
	return m.upsertExamQuestionFn(ctx, in)
}

func (m *mockExamService) DeleteExamQuestion(ctx context.Context, examID, questionID int64) error {
	if m.deleteExamQuestionFn == nil {
		return errors.New("not implemented")
	}
	return m.deleteExamQuestionFn(ctx, examID, questionID)
}

func (m *mockExamService) GetAttemptSummary(ctx context.Context, attemptID int64) (*AttemptSummary, error) {
	if m.getAttemptSummaryFn == nil {
		return nil, errors.New("not implemented")
	}
	return m.getAttemptSummaryFn(ctx, attemptID)
}

func (m *mockExamService) GetAttemptQuestion(ctx context.Context, attemptID int64, questionNo int) (*AttemptQuestion, error) {
	if m.getAttemptQuestionFn == nil {
		return nil, errors.New("not implemented")
	}
	return m.getAttemptQuestionFn(ctx, attemptID, questionNo)
}

func (m *mockExamService) GetAttemptResult(ctx context.Context, attemptID int64) (*AttemptResult, error) {
	if m.getAttemptResultFn == nil {
		return nil, errors.New("not implemented")
	}
	return m.getAttemptResultFn(ctx, attemptID)
}

func (m *mockExamService) SaveAnswer(ctx context.Context, input SaveAnswerInput) error {
	if m.saveAnswerFn == nil {
		return errors.New("not implemented")
	}
	return m.saveAnswerFn(ctx, input)
}

func (m *mockExamService) SubmitAttempt(ctx context.Context, attemptID int64) (*AttemptSummary, error) {
	if m.submitAttemptFn == nil {
		return nil, errors.New("not implemented")
	}
	return m.submitAttemptFn(ctx, attemptID)
}

func (m *mockExamService) GetAttemptOwner(ctx context.Context, attemptID int64) (int64, error) {
	if m.getAttemptOwnerFn == nil {
		return 0, errors.New("not implemented")
	}
	return m.getAttemptOwnerFn(ctx, attemptID)
}

func (m *mockExamService) LogAttemptEvent(ctx context.Context, input AttemptEventInput) (*AttemptEvent, error) {
	if m.logAttemptEventFn == nil {
		return nil, errors.New("not implemented")
	}
	return m.logAttemptEventFn(ctx, input)
}

func (m *mockExamService) ListAttemptEvents(ctx context.Context, attemptID int64, limit int) ([]AttemptEvent, error) {
	if m.listAttemptEventsFn == nil {
		return nil, errors.New("not implemented")
	}
	return m.listAttemptEventsFn(ctx, attemptID, limit)
}

func withChiParam(r *http.Request, key, value string) *http.Request {
	rctx := chi.RouteContext(r.Context())
	if rctx == nil {
		rctx = chi.NewRouteContext()
	}
	rctx.URLParams.Add(key, value)
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}

func decodeBody(t *testing.T, rr *httptest.ResponseRecorder) map[string]interface{} {
	t.Helper()
	var out map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return out
}

func TestGetAttemptForbiddenForNonOwner(t *testing.T) {
	calledSummary := false
	h := NewHandler(&mockExamService{
		getAttemptOwnerFn: func(ctx context.Context, attemptID int64) (int64, error) { return 99, nil },
		getAttemptSummaryFn: func(ctx context.Context, attemptID int64) (*AttemptSummary, error) {
			calledSummary = true
			return &AttemptSummary{ID: attemptID}, nil
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/attempts/10", nil)
	req = withChiParam(req, "id", "10")
	req = req.WithContext(auth.ContextWithUser(req.Context(), &auth.User{ID: 1, Role: "siswa"}))
	w := httptest.NewRecorder()

	h.GetAttempt(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", w.Code)
	}
	if calledSummary {
		t.Fatalf("expected summary not called when forbidden")
	}
}

func TestGetAttemptAllowedForAdmin(t *testing.T) {
	calledOwner := false
	calledSummary := false
	h := NewHandler(&mockExamService{
		getAttemptOwnerFn: func(ctx context.Context, attemptID int64) (int64, error) {
			calledOwner = true
			return 99, nil
		},
		getAttemptSummaryFn: func(ctx context.Context, attemptID int64) (*AttemptSummary, error) {
			calledSummary = true
			return &AttemptSummary{ID: attemptID, Status: "in_progress", ExpiresAt: time.Now().Add(time.Minute)}, nil
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/attempts/11", nil)
	req = withChiParam(req, "id", "11")
	req = req.WithContext(auth.ContextWithUser(req.Context(), &auth.User{ID: 7, Role: "admin"}))
	w := httptest.NewRecorder()

	h.GetAttempt(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if calledOwner {
		t.Fatalf("owner lookup should be skipped for admin/proktor")
	}
	if !calledSummary {
		t.Fatalf("summary should be called")
	}
}

func TestStartAttemptUsesSessionStudentIDForNonPrivileged(t *testing.T) {
	var gotExamID int64
	var gotStudentID int64
	h := NewHandler(&mockExamService{
		startAttemptFn: func(ctx context.Context, examID, studentID int64, examToken string) (*Attempt, error) {
			gotExamID = examID
			gotStudentID = studentID
			return &Attempt{ID: 1, ExamID: examID, StudentID: studentID, Status: "in_progress", StartedAt: time.Now(), ExpiresAt: time.Now().Add(time.Hour)}, nil
		},
	})

	payload := []byte(`{"exam_id":2}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/attempts/start", bytes.NewReader(payload))
	req = req.WithContext(auth.ContextWithUser(req.Context(), &auth.User{ID: 15, Role: "siswa"}))
	w := httptest.NewRecorder()

	h.Start(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if gotExamID != 2 {
		t.Fatalf("expected exam_id=2, got %d", gotExamID)
	}
	if gotStudentID != 15 {
		t.Fatalf("expected student_id forced to 15, got %d", gotStudentID)
	}
}

func TestStartAttemptAdminRequiresStudentID(t *testing.T) {
	h := NewHandler(&mockExamService{
		startAttemptFn: func(ctx context.Context, examID, studentID int64, examToken string) (*Attempt, error) {
			return &Attempt{ID: 1}, nil
		},
	})

	payload := []byte(`{"exam_id":2}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/attempts/start", bytes.NewReader(payload))
	req = req.WithContext(auth.ContextWithUser(req.Context(), &auth.User{ID: 7, Role: "admin"}))
	w := httptest.NewRecorder()

	h.Start(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
	body := decodeBody(t, w)
	if body["error"] == nil {
		t.Fatalf("expected error message")
	}
}

func TestSaveAnswerForbiddenForNonOwner(t *testing.T) {
	saveCalled := false
	h := NewHandler(&mockExamService{
		getAttemptOwnerFn: func(ctx context.Context, attemptID int64) (int64, error) { return 88, nil },
		saveAnswerFn: func(ctx context.Context, input SaveAnswerInput) error {
			saveCalled = true
			return nil
		},
	})

	payload := []byte(`{"answer_payload":{"selected":["A"]},"is_doubt":false}`)
	req := httptest.NewRequest(http.MethodPut, "/api/v1/attempts/12/answers/10", bytes.NewReader(payload))
	req = withChiParam(req, "id", "12")
	req = withChiParam(req, "questionID", "10")
	req = req.WithContext(auth.ContextWithUser(req.Context(), &auth.User{ID: 1, Role: "siswa"}))
	w := httptest.NewRecorder()

	h.SaveAnswer(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", w.Code)
	}
	if saveCalled {
		t.Fatalf("save should not be called for non-owner")
	}
}

func TestSubmitIdempotentReturnsSameSummary(t *testing.T) {
	fixedSubmittedAt := time.Now()
	fixedSummary := &AttemptSummary{
		ID:              55,
		ExamID:          1,
		StudentID:       2,
		Status:          "submitted",
		StartedAt:       time.Now().Add(-time.Hour),
		ExpiresAt:       time.Now().Add(10 * time.Minute),
		SubmittedAt:     &fixedSubmittedAt,
		RemainingSecs:   0,
		TotalQuestions:  1,
		Answered:        1,
		Doubt:           0,
		TotalCorrect:    1,
		TotalWrong:      0,
		TotalUnanswered: 0,
		Score:           1,
	}

	submitCalls := 0
	h := NewHandler(&mockExamService{
		getAttemptOwnerFn: func(ctx context.Context, attemptID int64) (int64, error) { return 2, nil },
		submitAttemptFn: func(ctx context.Context, attemptID int64) (*AttemptSummary, error) {
			submitCalls++
			return fixedSummary, nil
		},
	})

	callSubmit := func() map[string]interface{} {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/attempts/55/submit", nil)
		req = withChiParam(req, "id", "55")
		req = req.WithContext(auth.ContextWithUser(req.Context(), &auth.User{ID: 2, Role: "siswa"}))
		w := httptest.NewRecorder()
		h.Submit(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}
		return decodeBody(t, w)
	}

	first := callSubmit()
	second := callSubmit()

	if submitCalls != 2 {
		t.Fatalf("expected submit called twice, got %d", submitCalls)
	}
	firstData, _ := json.Marshal(first["data"])
	secondData, _ := json.Marshal(second["data"])
	if string(firstData) != string(secondData) {
		t.Fatalf("expected same summary on repeated submit, got different responses")
	}
}

func TestResultPolicyDenied(t *testing.T) {
	h := NewHandler(&mockExamService{
		getAttemptOwnerFn: func(ctx context.Context, attemptID int64) (int64, error) { return 2, nil },
		getAttemptResultFn: func(ctx context.Context, attemptID int64) (*AttemptResult, error) {
			return nil, ErrResultPolicyDenied
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/attempts/55/result", nil)
	req = withChiParam(req, "id", "55")
	req = req.WithContext(auth.ContextWithUser(req.Context(), &auth.User{ID: 2, Role: "siswa"}))
	w := httptest.NewRecorder()

	h.Result(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", w.Code)
	}
}

func TestLogEventForbiddenForNonOwner(t *testing.T) {
	logCalled := false
	h := NewHandler(&mockExamService{
		getAttemptOwnerFn: func(ctx context.Context, attemptID int64) (int64, error) { return 99, nil },
		logAttemptEventFn: func(ctx context.Context, input AttemptEventInput) (*AttemptEvent, error) {
			logCalled = true
			return &AttemptEvent{ID: 1, AttemptID: input.AttemptID, EventType: input.EventType}, nil
		},
	})

	payload := []byte(`{"event_type":"tab_blur","payload":{"count":1}}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/attempts/77/events", bytes.NewReader(payload))
	req = withChiParam(req, "id", "77")
	req = req.WithContext(auth.ContextWithUser(req.Context(), &auth.User{ID: 1, Role: "siswa"}))
	w := httptest.NewRecorder()

	h.LogEvent(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", w.Code)
	}
	if logCalled {
		t.Fatalf("log event should not be called for non-owner")
	}
}

func TestListEventsOK(t *testing.T) {
	h := NewHandler(&mockExamService{
		listAttemptEventsFn: func(ctx context.Context, attemptID int64, limit int) ([]AttemptEvent, error) {
			if attemptID != 55 {
				t.Fatalf("unexpected attempt id: %d", attemptID)
			}
			return []AttemptEvent{{ID: 1, AttemptID: 55, EventType: "tab_blur"}}, nil
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/attempts/55/events?limit=100", nil)
	req = withChiParam(req, "id", "55")
	w := httptest.NewRecorder()

	h.ListEvents(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}
