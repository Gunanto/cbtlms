package question

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"cbtlms/internal/auth"

	"github.com/go-chi/chi/v5"
)

type mockQuestionService struct {
	createQuestionFn     func(ctx context.Context, in CreateQuestionInput) (*QuestionBlueprint, error)
	listQuestionsFn      func(ctx context.Context, subjectID int64, ownerID *int64) ([]QuestionBlueprint, error)
	createFn             func(ctx context.Context, in CreateStimulusInput) (*Stimulus, error)
	listFn               func(ctx context.Context, subjectID int64) ([]Stimulus, error)
	updateStimulusFn     func(ctx context.Context, in UpdateStimulusInput) (*Stimulus, error)
	deleteStimulusFn     func(ctx context.Context, stimulusID int64) error
	createVersionFn      func(ctx context.Context, in CreateQuestionVersionInput) (*QuestionVersion, error)
	updateVersionFn      func(ctx context.Context, in UpdateQuestionVersionInput) (*QuestionVersion, error)
	deleteVersionFn      func(ctx context.Context, questionID int64, versionNo int) error
	finalizeFn           func(ctx context.Context, questionID int64, versionNo int) (*QuestionVersion, error)
	listVersionsFn       func(ctx context.Context, questionID int64) ([]QuestionVersion, error)
	createParallelFn     func(ctx context.Context, in CreateQuestionParallelInput) (*QuestionParallel, error)
	listParallelsFn      func(ctx context.Context, examID int64, parallelGroup string) ([]QuestionParallel, error)
	updateParallelFn     func(ctx context.Context, in UpdateQuestionParallelInput) (*QuestionParallel, error)
	deleteParallelFn     func(ctx context.Context, examID, parallelID int64) error
	createReviewTaskFn   func(ctx context.Context, in CreateReviewTaskInput) (*ReviewTask, error)
	decideReviewTaskFn   func(ctx context.Context, in ReviewDecisionInput) (*ReviewTask, error)
	listReviewTasksFn    func(ctx context.Context, reviewerID int64, status string) ([]ReviewTask, error)
	getQuestionReviewsFn func(ctx context.Context, questionID int64) ([]QuestionReview, error)
}

func (m *mockQuestionService) CreateQuestion(ctx context.Context, in CreateQuestionInput) (*QuestionBlueprint, error) {
	if m.createQuestionFn == nil {
		return nil, errors.New("not implemented")
	}
	return m.createQuestionFn(ctx, in)
}

func (m *mockQuestionService) ListQuestions(ctx context.Context, subjectID int64, ownerID *int64) ([]QuestionBlueprint, error) {
	if m.listQuestionsFn == nil {
		return nil, errors.New("not implemented")
	}
	return m.listQuestionsFn(ctx, subjectID, ownerID)
}

func (m *mockQuestionService) CreateStimulus(ctx context.Context, in CreateStimulusInput) (*Stimulus, error) {
	if m.createFn == nil {
		return nil, errors.New("not implemented")
	}
	return m.createFn(ctx, in)
}

func (m *mockQuestionService) ListStimuliBySubject(ctx context.Context, subjectID int64) ([]Stimulus, error) {
	if m.listFn == nil {
		return nil, errors.New("not implemented")
	}
	return m.listFn(ctx, subjectID)
}

func (m *mockQuestionService) UpdateStimulus(ctx context.Context, in UpdateStimulusInput) (*Stimulus, error) {
	if m.updateStimulusFn == nil {
		return nil, errors.New("not implemented")
	}
	return m.updateStimulusFn(ctx, in)
}

func (m *mockQuestionService) DeleteStimulus(ctx context.Context, stimulusID int64) error {
	if m.deleteStimulusFn == nil {
		return errors.New("not implemented")
	}
	return m.deleteStimulusFn(ctx, stimulusID)
}

func (m *mockQuestionService) CreateQuestionVersion(ctx context.Context, in CreateQuestionVersionInput) (*QuestionVersion, error) {
	if m.createVersionFn == nil {
		return nil, errors.New("not implemented")
	}
	return m.createVersionFn(ctx, in)
}

func (m *mockQuestionService) UpdateQuestionVersion(ctx context.Context, in UpdateQuestionVersionInput) (*QuestionVersion, error) {
	if m.updateVersionFn == nil {
		return nil, errors.New("not implemented")
	}
	return m.updateVersionFn(ctx, in)
}

func (m *mockQuestionService) DeleteQuestionVersion(ctx context.Context, questionID int64, versionNo int) error {
	if m.deleteVersionFn == nil {
		return errors.New("not implemented")
	}
	return m.deleteVersionFn(ctx, questionID, versionNo)
}

func (m *mockQuestionService) FinalizeQuestionVersion(ctx context.Context, questionID int64, versionNo int) (*QuestionVersion, error) {
	if m.finalizeFn == nil {
		return nil, errors.New("not implemented")
	}
	return m.finalizeFn(ctx, questionID, versionNo)
}

func (m *mockQuestionService) ListQuestionVersions(ctx context.Context, questionID int64) ([]QuestionVersion, error) {
	if m.listVersionsFn == nil {
		return nil, errors.New("not implemented")
	}
	return m.listVersionsFn(ctx, questionID)
}

func (m *mockQuestionService) CreateQuestionParallel(ctx context.Context, in CreateQuestionParallelInput) (*QuestionParallel, error) {
	if m.createParallelFn == nil {
		return nil, errors.New("not implemented")
	}
	return m.createParallelFn(ctx, in)
}

func (m *mockQuestionService) ListQuestionParallels(ctx context.Context, examID int64, parallelGroup string) ([]QuestionParallel, error) {
	if m.listParallelsFn == nil {
		return nil, errors.New("not implemented")
	}
	return m.listParallelsFn(ctx, examID, parallelGroup)
}

func (m *mockQuestionService) UpdateQuestionParallel(ctx context.Context, in UpdateQuestionParallelInput) (*QuestionParallel, error) {
	if m.updateParallelFn == nil {
		return nil, errors.New("not implemented")
	}
	return m.updateParallelFn(ctx, in)
}

func (m *mockQuestionService) DeleteQuestionParallel(ctx context.Context, examID, parallelID int64) error {
	if m.deleteParallelFn == nil {
		return errors.New("not implemented")
	}
	return m.deleteParallelFn(ctx, examID, parallelID)
}

func (m *mockQuestionService) CreateReviewTask(ctx context.Context, in CreateReviewTaskInput) (*ReviewTask, error) {
	if m.createReviewTaskFn == nil {
		return nil, errors.New("not implemented")
	}
	return m.createReviewTaskFn(ctx, in)
}

func (m *mockQuestionService) DecideReviewTask(ctx context.Context, in ReviewDecisionInput) (*ReviewTask, error) {
	if m.decideReviewTaskFn == nil {
		return nil, errors.New("not implemented")
	}
	return m.decideReviewTaskFn(ctx, in)
}

func (m *mockQuestionService) ListReviewTasks(ctx context.Context, reviewerID int64, status string) ([]ReviewTask, error) {
	if m.listReviewTasksFn == nil {
		return nil, errors.New("not implemented")
	}
	return m.listReviewTasksFn(ctx, reviewerID, status)
}

func (m *mockQuestionService) GetQuestionReviews(ctx context.Context, questionID int64) ([]QuestionReview, error) {
	if m.getQuestionReviewsFn == nil {
		return nil, errors.New("not implemented")
	}
	return m.getQuestionReviewsFn(ctx, questionID)
}

func decodeMap(t *testing.T, rr *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var out map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return out
}

func withParam(r *http.Request, key, value string) *http.Request {
	rctx := chi.RouteContext(r.Context())
	if rctx == nil {
		rctx = chi.NewRouteContext()
	}
	rctx.URLParams.Add(key, value)
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}

func TestListQuestionsGuruDefaultsToOwnerOnly(t *testing.T) {
	h := &Handler{svc: &mockQuestionService{
		listQuestionsFn: func(ctx context.Context, subjectID int64, ownerID *int64) ([]QuestionBlueprint, error) {
			if subjectID != 3 {
				t.Fatalf("unexpected subject id: %d", subjectID)
			}
			if ownerID == nil || *ownerID != 9 {
				t.Fatalf("expected ownerID=9, got %+v", ownerID)
			}
			return []QuestionBlueprint{{ID: 101, SubjectID: 3}}, nil
		},
	}}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/questions?subject_id=3", nil)
	req = req.WithContext(auth.ContextWithUser(req.Context(), &auth.User{ID: 9, Role: "guru"}))
	w := httptest.NewRecorder()

	h.ListQuestions(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestListQuestionsAdminDefaultsAll(t *testing.T) {
	h := &Handler{svc: &mockQuestionService{
		listQuestionsFn: func(ctx context.Context, subjectID int64, ownerID *int64) ([]QuestionBlueprint, error) {
			if ownerID != nil {
				t.Fatalf("expected ownerID nil, got %+v", ownerID)
			}
			return []QuestionBlueprint{{ID: 201, SubjectID: 1}}, nil
		},
	}}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/questions", nil)
	req = req.WithContext(auth.ContextWithUser(req.Context(), &auth.User{ID: 1, Role: "admin"}))
	w := httptest.NewRecorder()

	h.ListQuestions(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestListQuestionsOwnerOnlyQueryFalseForGuru(t *testing.T) {
	h := &Handler{svc: &mockQuestionService{
		listQuestionsFn: func(ctx context.Context, subjectID int64, ownerID *int64) ([]QuestionBlueprint, error) {
			if ownerID != nil {
				t.Fatalf("expected ownerID nil when owner_only=false, got %+v", ownerID)
			}
			return []QuestionBlueprint{{ID: 301, SubjectID: 2}}, nil
		},
	}}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/questions?owner_only=false", nil)
	req = req.WithContext(auth.ContextWithUser(req.Context(), &auth.User{ID: 9, Role: "guru"}))
	w := httptest.NewRecorder()

	h.ListQuestions(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestListStimuliRequiresSubjectID(t *testing.T) {
	h := &Handler{svc: &mockQuestionService{}}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/stimuli", nil)
	w := httptest.NewRecorder()

	h.ListStimuli(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestCreateStimulusOK(t *testing.T) {
	h := &Handler{svc: &mockQuestionService{
		createFn: func(ctx context.Context, in CreateStimulusInput) (*Stimulus, error) {
			if in.CreatedBy != 9 || in.SubjectID != 1 || in.StimulusType != "single" {
				t.Fatalf("unexpected input: %+v", in)
			}
			return &Stimulus{
				ID:           11,
				SubjectID:    in.SubjectID,
				Title:        in.Title,
				StimulusType: in.StimulusType,
				Content:      in.Content,
				IsActive:     true,
			}, nil
		},
	}}

	payload := []byte(`{"subject_id":1,"title":"Stimulus A","stimulus_type":"single","content":{"body":"Teks"}}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/stimuli", bytes.NewReader(payload))
	req = req.WithContext(auth.ContextWithUser(req.Context(), &auth.User{ID: 9, Role: "guru"}))
	w := httptest.NewRecorder()

	h.CreateStimulus(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", w.Code)
	}
	body := decodeMap(t, w)
	if body["ok"] != true {
		t.Fatalf("expected ok=true")
	}
}

func TestCreateStimulusSubjectNotFound(t *testing.T) {
	h := &Handler{svc: &mockQuestionService{
		createFn: func(ctx context.Context, in CreateStimulusInput) (*Stimulus, error) {
			return nil, ErrSubjectNotFound
		},
	}}

	payload := []byte(`{"subject_id":99,"title":"Stimulus","stimulus_type":"single","content":{"body":"Teks"}}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/stimuli", bytes.NewReader(payload))
	req = req.WithContext(auth.ContextWithUser(req.Context(), &auth.User{ID: 9, Role: "guru"}))
	w := httptest.NewRecorder()

	h.CreateStimulus(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestCreateQuestionVersionOK(t *testing.T) {
	h := &Handler{svc: &mockQuestionService{
		createVersionFn: func(ctx context.Context, in CreateQuestionVersionInput) (*QuestionVersion, error) {
			if in.QuestionID != 5 || in.CreatedBy != 10 {
				t.Fatalf("unexpected input: %+v", in)
			}
			return &QuestionVersion{ID: 100, QuestionID: 5, VersionNo: 2, Status: "draft"}, nil
		},
	}}

	payload := []byte(`{"stem_html":"<p>baru</p>","answer_key":{"selected":["A"]}}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/questions/5/versions", bytes.NewReader(payload))
	req = withParam(req, "id", "5")
	req = req.WithContext(auth.ContextWithUser(req.Context(), &auth.User{ID: 10, Role: "guru"}))
	w := httptest.NewRecorder()

	h.CreateQuestionVersion(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", w.Code)
	}
}

func TestFinalizeQuestionVersionNotFound(t *testing.T) {
	h := &Handler{svc: &mockQuestionService{
		finalizeFn: func(ctx context.Context, questionID int64, versionNo int) (*QuestionVersion, error) {
			return nil, ErrVersionNotFound
		},
	}}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/questions/5/versions/2/finalize", nil)
	req = withParam(req, "id", "5")
	req = withParam(req, "version", "2")
	w := httptest.NewRecorder()

	h.FinalizeQuestionVersion(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestUpdateQuestionVersionOK(t *testing.T) {
	h := &Handler{svc: &mockQuestionService{
		updateVersionFn: func(ctx context.Context, in UpdateQuestionVersionInput) (*QuestionVersion, error) {
			if in.QuestionID != 5 || in.VersionNo != 2 {
				t.Fatalf("unexpected input: %+v", in)
			}
			return &QuestionVersion{ID: 100, QuestionID: 5, VersionNo: 2, Status: "draft"}, nil
		},
	}}

	payload := []byte(`{"stem_html":"<p>ubah</p>","answer_key":{"correct":"A"}}`)
	req := httptest.NewRequest(http.MethodPut, "/api/v1/questions/5/versions/2", bytes.NewReader(payload))
	req = withParam(req, "id", "5")
	req = withParam(req, "version", "2")
	w := httptest.NewRecorder()

	h.UpdateQuestionVersion(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestDeleteQuestionVersionNotFound(t *testing.T) {
	h := &Handler{svc: &mockQuestionService{
		deleteVersionFn: func(ctx context.Context, questionID int64, versionNo int) error {
			return ErrVersionNotFound
		},
	}}

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/questions/5/versions/2", nil)
	req = withParam(req, "id", "5")
	req = withParam(req, "version", "2")
	w := httptest.NewRecorder()

	h.DeleteQuestionVersion(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestListQuestionVersionsOK(t *testing.T) {
	h := &Handler{svc: &mockQuestionService{
		listVersionsFn: func(ctx context.Context, questionID int64) ([]QuestionVersion, error) {
			return []QuestionVersion{{ID: 1, QuestionID: questionID, VersionNo: 2}, {ID: 2, QuestionID: questionID, VersionNo: 1}}, nil
		},
	}}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/questions/5/versions", nil)
	req = withParam(req, "id", "5")
	w := httptest.NewRecorder()

	h.ListQuestionVersions(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestCreateQuestionParallelOK(t *testing.T) {
	h := &Handler{svc: &mockQuestionService{
		createParallelFn: func(ctx context.Context, in CreateQuestionParallelInput) (*QuestionParallel, error) {
			if in.ExamID != 7 || in.QuestionID != 10 {
				t.Fatalf("unexpected input: %+v", in)
			}
			return &QuestionParallel{ID: 1, ExamID: in.ExamID, QuestionID: in.QuestionID, ParallelGroup: "default", ParallelOrder: 1, ParallelLabel: "pararel_1", IsActive: true}, nil
		},
	}}

	reqBody := []byte(`{"question_id":10,"parallel_group":"default","parallel_order":1,"parallel_label":"pararel_1"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/exams/7/parallels", bytes.NewReader(reqBody))
	req = withParam(req, "id", "7")
	w := httptest.NewRecorder()

	h.CreateQuestionParallel(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", w.Code)
	}
}

func TestListQuestionParallelsOK(t *testing.T) {
	h := &Handler{svc: &mockQuestionService{
		listParallelsFn: func(ctx context.Context, examID int64, parallelGroup string) ([]QuestionParallel, error) {
			return []QuestionParallel{{ID: 1, ExamID: examID, QuestionID: 10, ParallelGroup: "default", ParallelOrder: 1, ParallelLabel: "pararel_1"}}, nil
		},
	}}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/exams/7/parallels?parallel_group=default", nil)
	req = withParam(req, "id", "7")
	w := httptest.NewRecorder()

	h.ListQuestionParallels(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestUpdateQuestionParallelConflict(t *testing.T) {
	h := &Handler{svc: &mockQuestionService{
		updateParallelFn: func(ctx context.Context, in UpdateQuestionParallelInput) (*QuestionParallel, error) {
			return nil, ErrQuestionNotInExam
		},
	}}

	reqBody := []byte(`{"question_id":999,"parallel_group":"default","parallel_order":2,"parallel_label":"pararel_2"}`)
	req := httptest.NewRequest(http.MethodPut, "/api/v1/exams/7/parallels/1", bytes.NewReader(reqBody))
	req = withParam(req, "id", "7")
	req = withParam(req, "parallelID", "1")
	w := httptest.NewRecorder()

	h.UpdateQuestionParallel(w, req)
	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d", w.Code)
	}
}

func TestDeleteQuestionParallelNotFound(t *testing.T) {
	h := &Handler{svc: &mockQuestionService{
		deleteParallelFn: func(ctx context.Context, examID, parallelID int64) error {
			return ErrParallelNotFound
		},
	}}

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/exams/7/parallels/1", nil)
	req = withParam(req, "id", "7")
	req = withParam(req, "parallelID", "1")
	w := httptest.NewRecorder()

	h.DeleteQuestionParallel(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestCreateReviewTaskOK(t *testing.T) {
	h := &Handler{svc: &mockQuestionService{
		createReviewTaskFn: func(ctx context.Context, in CreateReviewTaskInput) (*ReviewTask, error) {
			if in.QuestionVersionID != 5 || in.ReviewerID != 77 {
				t.Fatalf("unexpected input: %+v", in)
			}
			return &ReviewTask{ID: 1, QuestionVersionID: 5, ReviewerID: 77, Status: "menunggu_reviu"}, nil
		},
	}}

	reqBody := []byte(`{"question_version_id":5,"reviewer_id":77,"note":"tolong telaah"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/reviews/tasks", bytes.NewReader(reqBody))
	w := httptest.NewRecorder()

	h.CreateReviewTask(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", w.Code)
	}
}

func TestDecideReviewTaskForbidden(t *testing.T) {
	h := &Handler{svc: &mockQuestionService{
		decideReviewTaskFn: func(ctx context.Context, in ReviewDecisionInput) (*ReviewTask, error) {
			return nil, ErrReviewForbidden
		},
	}}

	reqBody := []byte(`{"status":"disetujui","note":"ok"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/reviews/tasks/9/decision", bytes.NewReader(reqBody))
	req = withParam(req, "id", "9")
	req = req.WithContext(auth.ContextWithUser(req.Context(), &auth.User{ID: 8, Role: "guru"}))
	w := httptest.NewRecorder()

	h.DecideReviewTask(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", w.Code)
	}
}

func TestListReviewTasksForcesReviewerForGuru(t *testing.T) {
	h := &Handler{svc: &mockQuestionService{
		listReviewTasksFn: func(ctx context.Context, reviewerID int64, status string) ([]ReviewTask, error) {
			if reviewerID != 22 {
				t.Fatalf("expected reviewer_id forced to user id, got %d", reviewerID)
			}
			return []ReviewTask{}, nil
		},
	}}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/reviews/tasks?reviewer_id=999&status=menunggu_reviu", nil)
	req = req.WithContext(auth.ContextWithUser(req.Context(), &auth.User{ID: 22, Role: "guru"}))
	w := httptest.NewRecorder()

	h.ListReviewTasks(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestGetQuestionReviewsOK(t *testing.T) {
	h := &Handler{svc: &mockQuestionService{
		getQuestionReviewsFn: func(ctx context.Context, questionID int64) ([]QuestionReview, error) {
			return []QuestionReview{{Task: ReviewTask{ID: 1, QuestionID: questionID}}}, nil
		},
	}}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/questions/5/reviews", nil)
	req = withParam(req, "id", "5")
	w := httptest.NewRecorder()

	h.GetQuestionReviews(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}
