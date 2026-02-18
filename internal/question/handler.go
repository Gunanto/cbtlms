package question

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"cbtlms/internal/app/apiresp"
	"cbtlms/internal/auth"

	"github.com/go-chi/chi/v5"
)

type Handler struct {
	svc questionService
}

type questionService interface {
	CreateStimulus(ctx context.Context, in CreateStimulusInput) (*Stimulus, error)
	ListStimuliBySubject(ctx context.Context, subjectID int64) ([]Stimulus, error)
	CreateQuestionVersion(ctx context.Context, in CreateQuestionVersionInput) (*QuestionVersion, error)
	FinalizeQuestionVersion(ctx context.Context, questionID int64, versionNo int) (*QuestionVersion, error)
	ListQuestionVersions(ctx context.Context, questionID int64) ([]QuestionVersion, error)
	CreateQuestionParallel(ctx context.Context, in CreateQuestionParallelInput) (*QuestionParallel, error)
	ListQuestionParallels(ctx context.Context, examID int64, parallelGroup string) ([]QuestionParallel, error)
	UpdateQuestionParallel(ctx context.Context, in UpdateQuestionParallelInput) (*QuestionParallel, error)
	DeleteQuestionParallel(ctx context.Context, examID, parallelID int64) error
	CreateReviewTask(ctx context.Context, in CreateReviewTaskInput) (*ReviewTask, error)
	DecideReviewTask(ctx context.Context, in ReviewDecisionInput) (*ReviewTask, error)
	ListReviewTasks(ctx context.Context, reviewerID int64, status string) ([]ReviewTask, error)
	GetQuestionReviews(ctx context.Context, questionID int64) ([]QuestionReview, error)
}

type apiResponse struct {
	OK    bool        `json:"ok"`
	Data  interface{} `json:"data,omitempty"`
	Error string      `json:"error,omitempty"`
}

type createStimulusRequest struct {
	SubjectID    int64           `json:"subject_id"`
	Title        string          `json:"title"`
	StimulusType string          `json:"stimulus_type"`
	Content      json.RawMessage `json:"content"`
}

type createQuestionVersionRequest struct {
	StimulusID      *int64          `json:"stimulus_id"`
	StemHTML        *string         `json:"stem_html"`
	ExplanationHTML *string         `json:"explanation_html"`
	HintHTML        *string         `json:"hint_html"`
	AnswerKey       json.RawMessage `json:"answer_key"`
	DurationSeconds *int            `json:"duration_seconds"`
	Weight          *float64        `json:"weight"`
	ChangeNote      *string         `json:"change_note"`
}

type createQuestionParallelRequest struct {
	QuestionID    int64  `json:"question_id"`
	ParallelGroup string `json:"parallel_group"`
	ParallelOrder int    `json:"parallel_order"`
	ParallelLabel string `json:"parallel_label"`
}

type createReviewTaskRequest struct {
	QuestionVersionID int64  `json:"question_version_id"`
	ExamID            *int64 `json:"exam_id"`
	ReviewerID        int64  `json:"reviewer_id"`
	Note              string `json:"note"`
}

type reviewDecisionRequest struct {
	Status string `json:"status"`
	Note   string `json:"note"`
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) CreateStimulus(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.CurrentUser(r.Context())
	if !ok {
		writeJSON(w, r, http.StatusUnauthorized, apiResponse{OK: false, Error: "unauthorized"})
		return
	}

	var req createStimulusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, r, http.StatusBadRequest, apiResponse{OK: false, Error: "invalid request body"})
		return
	}

	item, err := h.svc.CreateStimulus(r.Context(), CreateStimulusInput{
		SubjectID:    req.SubjectID,
		Title:        req.Title,
		StimulusType: req.StimulusType,
		Content:      req.Content,
		CreatedBy:    user.ID,
	})
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidInput):
			writeJSON(w, r, http.StatusBadRequest, apiResponse{OK: false, Error: err.Error()})
		case errors.Is(err, ErrSubjectNotFound):
			writeJSON(w, r, http.StatusNotFound, apiResponse{OK: false, Error: err.Error()})
		default:
			writeJSON(w, r, http.StatusInternalServerError, apiResponse{OK: false, Error: "internal error"})
		}
		return
	}

	writeJSON(w, r, http.StatusCreated, apiResponse{OK: true, Data: item})
}

func (h *Handler) ListStimuli(w http.ResponseWriter, r *http.Request) {
	subjectIDRaw := strings.TrimSpace(r.URL.Query().Get("subject_id"))
	subjectID, err := strconv.ParseInt(subjectIDRaw, 10, 64)
	if err != nil || subjectID <= 0 {
		writeJSON(w, r, http.StatusBadRequest, apiResponse{OK: false, Error: "subject_id is required and must be positive"})
		return
	}

	items, err := h.svc.ListStimuliBySubject(r.Context(), subjectID)
	if err != nil {
		if errors.Is(err, ErrInvalidInput) {
			writeJSON(w, r, http.StatusBadRequest, apiResponse{OK: false, Error: err.Error()})
			return
		}
		writeJSON(w, r, http.StatusInternalServerError, apiResponse{OK: false, Error: "internal error"})
		return
	}

	writeJSON(w, r, http.StatusOK, apiResponse{OK: true, Data: items})
}

func (h *Handler) CreateQuestionVersion(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.CurrentUser(r.Context())
	if !ok {
		writeJSON(w, r, http.StatusUnauthorized, apiResponse{OK: false, Error: "unauthorized"})
		return
	}

	questionID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || questionID <= 0 {
		writeJSON(w, r, http.StatusBadRequest, apiResponse{OK: false, Error: "invalid question id"})
		return
	}

	var req createQuestionVersionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, r, http.StatusBadRequest, apiResponse{OK: false, Error: "invalid request body"})
		return
	}

	item, err := h.svc.CreateQuestionVersion(r.Context(), CreateQuestionVersionInput{
		QuestionID:      questionID,
		StimulusID:      req.StimulusID,
		StemHTML:        req.StemHTML,
		ExplanationHTML: req.ExplanationHTML,
		HintHTML:        req.HintHTML,
		AnswerKey:       req.AnswerKey,
		DurationSeconds: req.DurationSeconds,
		Weight:          req.Weight,
		ChangeNote:      req.ChangeNote,
		CreatedBy:       user.ID,
	})
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidInput):
			writeJSON(w, r, http.StatusBadRequest, apiResponse{OK: false, Error: err.Error()})
		case errors.Is(err, ErrQuestionNotFound):
			writeJSON(w, r, http.StatusNotFound, apiResponse{OK: false, Error: err.Error()})
		default:
			writeJSON(w, r, http.StatusInternalServerError, apiResponse{OK: false, Error: "internal error"})
		}
		return
	}

	writeJSON(w, r, http.StatusCreated, apiResponse{OK: true, Data: item})
}

func (h *Handler) FinalizeQuestionVersion(w http.ResponseWriter, r *http.Request) {
	questionID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || questionID <= 0 {
		writeJSON(w, r, http.StatusBadRequest, apiResponse{OK: false, Error: "invalid question id"})
		return
	}
	versionNo, err := strconv.Atoi(chi.URLParam(r, "version"))
	if err != nil || versionNo <= 0 {
		writeJSON(w, r, http.StatusBadRequest, apiResponse{OK: false, Error: "invalid version"})
		return
	}

	item, err := h.svc.FinalizeQuestionVersion(r.Context(), questionID, versionNo)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidInput):
			writeJSON(w, r, http.StatusBadRequest, apiResponse{OK: false, Error: err.Error()})
		case errors.Is(err, ErrVersionNotFound):
			writeJSON(w, r, http.StatusNotFound, apiResponse{OK: false, Error: err.Error()})
		default:
			writeJSON(w, r, http.StatusInternalServerError, apiResponse{OK: false, Error: "internal error"})
		}
		return
	}

	writeJSON(w, r, http.StatusOK, apiResponse{OK: true, Data: item})
}

func (h *Handler) ListQuestionVersions(w http.ResponseWriter, r *http.Request) {
	questionID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || questionID <= 0 {
		writeJSON(w, r, http.StatusBadRequest, apiResponse{OK: false, Error: "invalid question id"})
		return
	}

	items, err := h.svc.ListQuestionVersions(r.Context(), questionID)
	if err != nil {
		if errors.Is(err, ErrInvalidInput) {
			writeJSON(w, r, http.StatusBadRequest, apiResponse{OK: false, Error: err.Error()})
			return
		}
		writeJSON(w, r, http.StatusInternalServerError, apiResponse{OK: false, Error: "internal error"})
		return
	}

	writeJSON(w, r, http.StatusOK, apiResponse{OK: true, Data: items})
}

func (h *Handler) CreateQuestionParallel(w http.ResponseWriter, r *http.Request) {
	examID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || examID <= 0 {
		writeJSON(w, r, http.StatusBadRequest, apiResponse{OK: false, Error: "invalid exam id"})
		return
	}

	var req createQuestionParallelRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, r, http.StatusBadRequest, apiResponse{OK: false, Error: "invalid request body"})
		return
	}

	item, err := h.svc.CreateQuestionParallel(r.Context(), CreateQuestionParallelInput{
		ExamID:        examID,
		QuestionID:    req.QuestionID,
		ParallelGroup: req.ParallelGroup,
		ParallelOrder: req.ParallelOrder,
		ParallelLabel: req.ParallelLabel,
	})
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidInput):
			writeJSON(w, r, http.StatusBadRequest, apiResponse{OK: false, Error: err.Error()})
		case errors.Is(err, ErrExamNotFound):
			writeJSON(w, r, http.StatusNotFound, apiResponse{OK: false, Error: err.Error()})
		case errors.Is(err, ErrQuestionNotInExam):
			writeJSON(w, r, http.StatusConflict, apiResponse{OK: false, Error: err.Error()})
		default:
			writeJSON(w, r, http.StatusInternalServerError, apiResponse{OK: false, Error: "internal error"})
		}
		return
	}

	writeJSON(w, r, http.StatusCreated, apiResponse{OK: true, Data: item})
}

func (h *Handler) ListQuestionParallels(w http.ResponseWriter, r *http.Request) {
	examID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || examID <= 0 {
		writeJSON(w, r, http.StatusBadRequest, apiResponse{OK: false, Error: "invalid exam id"})
		return
	}
	parallelGroup := strings.TrimSpace(r.URL.Query().Get("parallel_group"))

	items, err := h.svc.ListQuestionParallels(r.Context(), examID, parallelGroup)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidInput):
			writeJSON(w, r, http.StatusBadRequest, apiResponse{OK: false, Error: err.Error()})
		case errors.Is(err, ErrExamNotFound):
			writeJSON(w, r, http.StatusNotFound, apiResponse{OK: false, Error: err.Error()})
		default:
			writeJSON(w, r, http.StatusInternalServerError, apiResponse{OK: false, Error: "internal error"})
		}
		return
	}
	writeJSON(w, r, http.StatusOK, apiResponse{OK: true, Data: items})
}

func (h *Handler) UpdateQuestionParallel(w http.ResponseWriter, r *http.Request) {
	examID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || examID <= 0 {
		writeJSON(w, r, http.StatusBadRequest, apiResponse{OK: false, Error: "invalid exam id"})
		return
	}
	parallelID, err := strconv.ParseInt(chi.URLParam(r, "parallelID"), 10, 64)
	if err != nil || parallelID <= 0 {
		writeJSON(w, r, http.StatusBadRequest, apiResponse{OK: false, Error: "invalid parallel id"})
		return
	}

	var req createQuestionParallelRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, r, http.StatusBadRequest, apiResponse{OK: false, Error: "invalid request body"})
		return
	}

	item, err := h.svc.UpdateQuestionParallel(r.Context(), UpdateQuestionParallelInput{
		ExamID:        examID,
		ParallelID:    parallelID,
		QuestionID:    req.QuestionID,
		ParallelGroup: req.ParallelGroup,
		ParallelOrder: req.ParallelOrder,
		ParallelLabel: req.ParallelLabel,
	})
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidInput):
			writeJSON(w, r, http.StatusBadRequest, apiResponse{OK: false, Error: err.Error()})
		case errors.Is(err, ErrParallelNotFound), errors.Is(err, ErrExamNotFound):
			writeJSON(w, r, http.StatusNotFound, apiResponse{OK: false, Error: err.Error()})
		case errors.Is(err, ErrQuestionNotInExam):
			writeJSON(w, r, http.StatusConflict, apiResponse{OK: false, Error: err.Error()})
		default:
			writeJSON(w, r, http.StatusInternalServerError, apiResponse{OK: false, Error: "internal error"})
		}
		return
	}
	writeJSON(w, r, http.StatusOK, apiResponse{OK: true, Data: item})
}

func (h *Handler) DeleteQuestionParallel(w http.ResponseWriter, r *http.Request) {
	examID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || examID <= 0 {
		writeJSON(w, r, http.StatusBadRequest, apiResponse{OK: false, Error: "invalid exam id"})
		return
	}
	parallelID, err := strconv.ParseInt(chi.URLParam(r, "parallelID"), 10, 64)
	if err != nil || parallelID <= 0 {
		writeJSON(w, r, http.StatusBadRequest, apiResponse{OK: false, Error: "invalid parallel id"})
		return
	}

	if err := h.svc.DeleteQuestionParallel(r.Context(), examID, parallelID); err != nil {
		switch {
		case errors.Is(err, ErrInvalidInput):
			writeJSON(w, r, http.StatusBadRequest, apiResponse{OK: false, Error: err.Error()})
		case errors.Is(err, ErrParallelNotFound):
			writeJSON(w, r, http.StatusNotFound, apiResponse{OK: false, Error: err.Error()})
		default:
			writeJSON(w, r, http.StatusInternalServerError, apiResponse{OK: false, Error: "internal error"})
		}
		return
	}

	writeJSON(w, r, http.StatusOK, apiResponse{OK: true, Data: map[string]string{"status": "deleted"}})
}

func (h *Handler) CreateReviewTask(w http.ResponseWriter, r *http.Request) {
	var req createReviewTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, r, http.StatusBadRequest, apiResponse{OK: false, Error: "invalid request body"})
		return
	}

	item, err := h.svc.CreateReviewTask(r.Context(), CreateReviewTaskInput{
		QuestionVersionID: req.QuestionVersionID,
		ExamID:            req.ExamID,
		ReviewerID:        req.ReviewerID,
		Note:              req.Note,
	})
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidInput):
			writeJSON(w, r, http.StatusBadRequest, apiResponse{OK: false, Error: err.Error()})
		case errors.Is(err, ErrVersionNotFound), errors.Is(err, ErrExamNotFound):
			writeJSON(w, r, http.StatusNotFound, apiResponse{OK: false, Error: err.Error()})
		default:
			writeJSON(w, r, http.StatusInternalServerError, apiResponse{OK: false, Error: "internal error"})
		}
		return
	}
	writeJSON(w, r, http.StatusCreated, apiResponse{OK: true, Data: item})
}

func (h *Handler) DecideReviewTask(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.CurrentUser(r.Context())
	if !ok {
		writeJSON(w, r, http.StatusUnauthorized, apiResponse{OK: false, Error: "unauthorized"})
		return
	}
	taskID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || taskID <= 0 {
		writeJSON(w, r, http.StatusBadRequest, apiResponse{OK: false, Error: "invalid review task id"})
		return
	}

	var req reviewDecisionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, r, http.StatusBadRequest, apiResponse{OK: false, Error: "invalid request body"})
		return
	}

	item, err := h.svc.DecideReviewTask(r.Context(), ReviewDecisionInput{
		TaskID:       taskID,
		ReviewerID:   user.ID,
		IsPrivileged: user.Role == "admin" || user.Role == "proktor",
		Status:       req.Status,
		Note:         req.Note,
	})
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidInput):
			writeJSON(w, r, http.StatusBadRequest, apiResponse{OK: false, Error: err.Error()})
		case errors.Is(err, ErrReviewTaskNotFound):
			writeJSON(w, r, http.StatusNotFound, apiResponse{OK: false, Error: err.Error()})
		case errors.Is(err, ErrReviewForbidden):
			writeJSON(w, r, http.StatusForbidden, apiResponse{OK: false, Error: "forbidden"})
		default:
			writeJSON(w, r, http.StatusInternalServerError, apiResponse{OK: false, Error: "internal error"})
		}
		return
	}
	writeJSON(w, r, http.StatusOK, apiResponse{OK: true, Data: item})
}

func (h *Handler) ListReviewTasks(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.CurrentUser(r.Context())
	if !ok {
		writeJSON(w, r, http.StatusUnauthorized, apiResponse{OK: false, Error: "unauthorized"})
		return
	}
	status := strings.TrimSpace(r.URL.Query().Get("status"))
	reviewerID := user.ID
	if user.Role == "admin" || user.Role == "proktor" {
		if raw := strings.TrimSpace(r.URL.Query().Get("reviewer_id")); raw != "" {
			if parsed, err := strconv.ParseInt(raw, 10, 64); err == nil && parsed > 0 {
				reviewerID = parsed
			}
		}
	}

	items, err := h.svc.ListReviewTasks(r.Context(), reviewerID, status)
	if err != nil {
		if errors.Is(err, ErrInvalidInput) {
			writeJSON(w, r, http.StatusBadRequest, apiResponse{OK: false, Error: err.Error()})
			return
		}
		writeJSON(w, r, http.StatusInternalServerError, apiResponse{OK: false, Error: "internal error"})
		return
	}
	writeJSON(w, r, http.StatusOK, apiResponse{OK: true, Data: items})
}

func (h *Handler) GetQuestionReviews(w http.ResponseWriter, r *http.Request) {
	questionID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || questionID <= 0 {
		writeJSON(w, r, http.StatusBadRequest, apiResponse{OK: false, Error: "invalid question id"})
		return
	}

	items, err := h.svc.GetQuestionReviews(r.Context(), questionID)
	if err != nil {
		if errors.Is(err, ErrInvalidInput) {
			writeJSON(w, r, http.StatusBadRequest, apiResponse{OK: false, Error: err.Error()})
			return
		}
		writeJSON(w, r, http.StatusInternalServerError, apiResponse{OK: false, Error: "internal error"})
		return
	}
	writeJSON(w, r, http.StatusOK, apiResponse{OK: true, Data: items})
}

func writeJSON(w http.ResponseWriter, r *http.Request, code int, payload apiResponse) {
	if payload.OK {
		apiresp.WriteOK(w, r, code, payload.Data)
		return
	}
	apiresp.WriteError(w, r, code, payload.Error)
}
