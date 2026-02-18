package exam

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"cbtlms/internal/auth"

	"github.com/go-chi/chi/v5"
)

type Handler struct {
	svc examService
}

type examService interface {
	StartAttempt(ctx context.Context, examID, studentID int64) (*Attempt, error)
	GetAttemptSummary(ctx context.Context, attemptID int64) (*AttemptSummary, error)
	GetAttemptResult(ctx context.Context, attemptID int64) (*AttemptResult, error)
	SaveAnswer(ctx context.Context, input SaveAnswerInput) error
	SubmitAttempt(ctx context.Context, attemptID int64) (*AttemptSummary, error)
	GetAttemptOwner(ctx context.Context, attemptID int64) (int64, error)
}

type response struct {
	OK    bool        `json:"ok"`
	Data  interface{} `json:"data,omitempty"`
	Error string      `json:"error,omitempty"`
}

type startAttemptRequest struct {
	ExamID    int64 `json:"exam_id"`
	StudentID int64 `json:"student_id"`
}

type saveAnswerRequest struct {
	AnswerPayload json.RawMessage `json:"answer_payload"`
	IsDoubt       bool            `json:"is_doubt"`
}

func NewHandler(svc examService) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Start(w http.ResponseWriter, r *http.Request) {
	var req startAttemptRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, response{OK: false, Error: "invalid request body"})
		return
	}
	if req.ExamID <= 0 {
		writeJSON(w, http.StatusBadRequest, response{OK: false, Error: "exam_id is required"})
		return
	}

	user, ok := auth.CurrentUser(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, response{OK: false, Error: "unauthorized"})
		return
	}

	isPrivileged := user.Role == "admin" || user.Role == "proktor"
	if isPrivileged {
		if req.StudentID <= 0 {
			writeJSON(w, http.StatusBadRequest, response{OK: false, Error: "student_id is required for admin/proktor"})
			return
		}
	} else {
		if req.StudentID > 0 && req.StudentID != user.ID {
			writeJSON(w, http.StatusForbidden, response{OK: false, Error: "forbidden"})
			return
		}
		req.StudentID = user.ID
	}

	attempt, err := h.svc.StartAttempt(r.Context(), req.ExamID, req.StudentID)
	if err != nil {
		switch {
		case errors.Is(err, ErrExamNotFound):
			writeJSON(w, http.StatusNotFound, response{OK: false, Error: err.Error()})
		default:
			writeJSON(w, http.StatusInternalServerError, response{OK: false, Error: "internal error"})
		}
		return
	}

	writeJSON(w, http.StatusOK, response{OK: true, Data: attempt})
}

func (h *Handler) GetAttempt(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.CurrentUser(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, response{OK: false, Error: "unauthorized"})
		return
	}

	attemptID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || attemptID <= 0 {
		writeJSON(w, http.StatusBadRequest, response{OK: false, Error: "invalid attempt id"})
		return
	}

	if err := h.authorizeAttemptAccess(r, user, attemptID); err != nil {
		switch {
		case errors.Is(err, ErrAttemptNotFound):
			writeJSON(w, http.StatusNotFound, response{OK: false, Error: err.Error()})
		case errors.Is(err, ErrAttemptForbidden):
			writeJSON(w, http.StatusForbidden, response{OK: false, Error: "forbidden"})
		default:
			writeJSON(w, http.StatusInternalServerError, response{OK: false, Error: "internal error"})
		}
		return
	}

	summary, err := h.svc.GetAttemptSummary(r.Context(), attemptID)
	if err != nil {
		switch {
		case errors.Is(err, ErrAttemptNotFound):
			writeJSON(w, http.StatusNotFound, response{OK: false, Error: err.Error()})
		default:
			writeJSON(w, http.StatusInternalServerError, response{OK: false, Error: "internal error"})
		}
		return
	}

	writeJSON(w, http.StatusOK, response{OK: true, Data: summary})
}

func (h *Handler) SaveAnswer(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.CurrentUser(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, response{OK: false, Error: "unauthorized"})
		return
	}

	attemptID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || attemptID <= 0 {
		writeJSON(w, http.StatusBadRequest, response{OK: false, Error: "invalid attempt id"})
		return
	}

	questionID, err := strconv.ParseInt(chi.URLParam(r, "questionID"), 10, 64)
	if err != nil || questionID <= 0 {
		writeJSON(w, http.StatusBadRequest, response{OK: false, Error: "invalid question id"})
		return
	}

	var req saveAnswerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, response{OK: false, Error: "invalid request body"})
		return
	}
	if len(req.AnswerPayload) == 0 {
		req.AnswerPayload = json.RawMessage(`{}`)
	}

	if err := h.authorizeAttemptAccess(r, user, attemptID); err != nil {
		switch {
		case errors.Is(err, ErrAttemptNotFound):
			writeJSON(w, http.StatusNotFound, response{OK: false, Error: err.Error()})
		case errors.Is(err, ErrAttemptForbidden):
			writeJSON(w, http.StatusForbidden, response{OK: false, Error: "forbidden"})
		default:
			writeJSON(w, http.StatusInternalServerError, response{OK: false, Error: "internal error"})
		}
		return
	}

	err = h.svc.SaveAnswer(r.Context(), SaveAnswerInput{
		AttemptID:     attemptID,
		QuestionID:    questionID,
		AnswerPayload: req.AnswerPayload,
		IsDoubt:       req.IsDoubt,
	})
	if err != nil {
		switch {
		case errors.Is(err, ErrAttemptNotFound):
			writeJSON(w, http.StatusNotFound, response{OK: false, Error: err.Error()})
		case errors.Is(err, ErrAttemptNotEditable), errors.Is(err, ErrQuestionNotInExam):
			writeJSON(w, http.StatusBadRequest, response{OK: false, Error: err.Error()})
		default:
			writeJSON(w, http.StatusInternalServerError, response{OK: false, Error: "internal error"})
		}
		return
	}

	writeJSON(w, http.StatusOK, response{OK: true, Data: map[string]string{"status": "saved"}})
}

func (h *Handler) Submit(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.CurrentUser(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, response{OK: false, Error: "unauthorized"})
		return
	}

	attemptID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || attemptID <= 0 {
		writeJSON(w, http.StatusBadRequest, response{OK: false, Error: "invalid attempt id"})
		return
	}

	if err := h.authorizeAttemptAccess(r, user, attemptID); err != nil {
		switch {
		case errors.Is(err, ErrAttemptNotFound):
			writeJSON(w, http.StatusNotFound, response{OK: false, Error: err.Error()})
		case errors.Is(err, ErrAttemptForbidden):
			writeJSON(w, http.StatusForbidden, response{OK: false, Error: "forbidden"})
		default:
			writeJSON(w, http.StatusInternalServerError, response{OK: false, Error: "internal error"})
		}
		return
	}

	summary, err := h.svc.SubmitAttempt(r.Context(), attemptID)
	if err != nil {
		switch {
		case errors.Is(err, ErrAttemptNotFound):
			writeJSON(w, http.StatusNotFound, response{OK: false, Error: err.Error()})
		default:
			writeJSON(w, http.StatusInternalServerError, response{OK: false, Error: "internal error"})
		}
		return
	}

	writeJSON(w, http.StatusOK, response{OK: true, Data: summary})
}

func (h *Handler) Result(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.CurrentUser(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, response{OK: false, Error: "unauthorized"})
		return
	}

	attemptID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || attemptID <= 0 {
		writeJSON(w, http.StatusBadRequest, response{OK: false, Error: "invalid attempt id"})
		return
	}

	if err := h.authorizeAttemptAccess(r, user, attemptID); err != nil {
		switch {
		case errors.Is(err, ErrAttemptNotFound):
			writeJSON(w, http.StatusNotFound, response{OK: false, Error: err.Error()})
		case errors.Is(err, ErrAttemptForbidden):
			writeJSON(w, http.StatusForbidden, response{OK: false, Error: "forbidden"})
		default:
			writeJSON(w, http.StatusInternalServerError, response{OK: false, Error: "internal error"})
		}
		return
	}

	result, err := h.svc.GetAttemptResult(r.Context(), attemptID)
	if err != nil {
		switch {
		case errors.Is(err, ErrAttemptNotFound):
			writeJSON(w, http.StatusNotFound, response{OK: false, Error: err.Error()})
		case errors.Is(err, ErrAttemptNotFinal):
			writeJSON(w, http.StatusBadRequest, response{OK: false, Error: err.Error()})
		case errors.Is(err, ErrResultPolicyDenied):
			writeJSON(w, http.StatusForbidden, response{OK: false, Error: err.Error()})
		default:
			writeJSON(w, http.StatusInternalServerError, response{OK: false, Error: "internal error"})
		}
		return
	}

	writeJSON(w, http.StatusOK, response{OK: true, Data: result})
}

func writeJSON(w http.ResponseWriter, code int, payload response) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(payload)
}

func (h *Handler) authorizeAttemptAccess(r *http.Request, user *auth.User, attemptID int64) error {
	if user.Role == "admin" || user.Role == "proktor" {
		return nil
	}

	ownerID, err := h.svc.GetAttemptOwner(r.Context(), attemptID)
	if err != nil {
		return err
	}
	if ownerID != user.ID {
		return ErrAttemptForbidden
	}
	return nil
}
