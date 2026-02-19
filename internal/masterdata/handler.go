package masterdata

import (
	"database/sql"
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
	svc *Service
}

type apiResponse struct {
	OK    bool        `json:"ok"`
	Data  interface{} `json:"data,omitempty"`
	Error string      `json:"error,omitempty"`
}

type createSchoolRequest struct {
	Name    string `json:"name"`
	Code    string `json:"code"`
	Address string `json:"address"`
}

type createClassRequest struct {
	SchoolID   int64  `json:"school_id"`
	Name       string `json:"name"`
	GradeLevel string `json:"grade_level"`
}

type createEducationLevelRequest struct {
	Name string `json:"name"`
}

type updateEducationLevelRequest struct {
	Name string `json:"name"`
}

type updateSchoolRequest struct {
	Name    string `json:"name"`
	Code    string `json:"code"`
	Address string `json:"address"`
}

type updateClassRequest struct {
	SchoolID   int64  `json:"school_id"`
	Name       string `json:"name"`
	GradeLevel string `json:"grade_level"`
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) ListEducationLevels(w http.ResponseWriter, r *http.Request) {
	activeOnly := true
	if strings.TrimSpace(r.URL.Query().Get("all")) == "1" {
		activeOnly = false
	}

	items, err := h.svc.ListEducationLevels(r.Context(), activeOnly)
	if err != nil {
		writeJSON(w, r, http.StatusInternalServerError, apiResponse{OK: false, Error: "internal error"})
		return
	}
	writeJSON(w, r, http.StatusOK, apiResponse{OK: true, Data: items})
}

func (h *Handler) ListSchools(w http.ResponseWriter, r *http.Request) {
	activeOnly := true
	if strings.TrimSpace(r.URL.Query().Get("all")) == "1" {
		activeOnly = false
	}

	items, err := h.svc.ListSchools(r.Context(), activeOnly)
	if err != nil {
		writeJSON(w, r, http.StatusInternalServerError, apiResponse{OK: false, Error: "internal error"})
		return
	}
	writeJSON(w, r, http.StatusOK, apiResponse{OK: true, Data: items})
}

func (h *Handler) ListClasses(w http.ResponseWriter, r *http.Request) {
	activeOnly := true
	if strings.TrimSpace(r.URL.Query().Get("all")) == "1" {
		activeOnly = false
	}

	schoolID := int64(0)
	if v := strings.TrimSpace(r.URL.Query().Get("school_id")); v != "" {
		id, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			writeJSON(w, r, http.StatusBadRequest, apiResponse{OK: false, Error: "invalid school_id"})
			return
		}
		schoolID = id
	}

	items, err := h.svc.ListClasses(r.Context(), schoolID, activeOnly)
	if err != nil {
		writeJSON(w, r, http.StatusInternalServerError, apiResponse{OK: false, Error: "internal error"})
		return
	}
	writeJSON(w, r, http.StatusOK, apiResponse{OK: true, Data: items})
}

func (h *Handler) CreateEducationLevel(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.CurrentUser(r.Context())
	if !ok {
		writeJSON(w, r, http.StatusUnauthorized, apiResponse{OK: false, Error: "unauthorized"})
		return
	}

	var req createEducationLevelRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, r, http.StatusBadRequest, apiResponse{OK: false, Error: "invalid request body"})
		return
	}

	item, err := h.svc.CreateEducationLevel(r.Context(), user.ID, req.Name)
	if err != nil {
		if errors.Is(err, ErrInvalidInput) {
			writeJSON(w, r, http.StatusBadRequest, apiResponse{OK: false, Error: "name is required"})
			return
		}
		writeJSON(w, r, http.StatusBadRequest, apiResponse{OK: false, Error: err.Error()})
		return
	}
	writeJSON(w, r, http.StatusCreated, apiResponse{OK: true, Data: item})
}

func (h *Handler) UpdateEducationLevel(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.CurrentUser(r.Context())
	if !ok {
		writeJSON(w, r, http.StatusUnauthorized, apiResponse{OK: false, Error: "unauthorized"})
		return
	}

	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		writeJSON(w, r, http.StatusBadRequest, apiResponse{OK: false, Error: "invalid id"})
		return
	}

	var req updateEducationLevelRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, r, http.StatusBadRequest, apiResponse{OK: false, Error: "invalid request body"})
		return
	}

	item, err := h.svc.UpdateEducationLevel(r.Context(), user.ID, id, req.Name)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidInput):
			writeJSON(w, r, http.StatusBadRequest, apiResponse{OK: false, Error: "id and name are required"})
		case errors.Is(err, sql.ErrNoRows):
			writeJSON(w, r, http.StatusNotFound, apiResponse{OK: false, Error: "education level not found"})
		default:
			writeJSON(w, r, http.StatusBadRequest, apiResponse{OK: false, Error: err.Error()})
		}
		return
	}
	writeJSON(w, r, http.StatusOK, apiResponse{OK: true, Data: item})
}

func (h *Handler) DeleteEducationLevel(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.CurrentUser(r.Context())
	if !ok {
		writeJSON(w, r, http.StatusUnauthorized, apiResponse{OK: false, Error: "unauthorized"})
		return
	}

	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		writeJSON(w, r, http.StatusBadRequest, apiResponse{OK: false, Error: "invalid id"})
		return
	}

	err = h.svc.DeleteEducationLevel(r.Context(), user.ID, id)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidInput):
			writeJSON(w, r, http.StatusBadRequest, apiResponse{OK: false, Error: "invalid id"})
		case errors.Is(err, sql.ErrNoRows):
			writeJSON(w, r, http.StatusNotFound, apiResponse{OK: false, Error: "education level not found"})
		default:
			writeJSON(w, r, http.StatusBadRequest, apiResponse{OK: false, Error: err.Error()})
		}
		return
	}

	writeJSON(w, r, http.StatusOK, apiResponse{OK: true, Data: map[string]string{"status": "deleted"}})
}

func (h *Handler) CreateSchool(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.CurrentUser(r.Context())
	if !ok {
		writeJSON(w, r, http.StatusUnauthorized, apiResponse{OK: false, Error: "unauthorized"})
		return
	}

	var req createSchoolRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, r, http.StatusBadRequest, apiResponse{OK: false, Error: "invalid request body"})
		return
	}

	school, err := h.svc.CreateSchool(r.Context(), user.ID, CreateSchoolInput{
		Name:    req.Name,
		Code:    req.Code,
		Address: req.Address,
	})
	if err != nil {
		if errors.Is(err, ErrInvalidInput) {
			writeJSON(w, r, http.StatusBadRequest, apiResponse{OK: false, Error: "name is required"})
			return
		}
		writeJSON(w, r, http.StatusBadRequest, apiResponse{OK: false, Error: err.Error()})
		return
	}

	writeJSON(w, r, http.StatusCreated, apiResponse{OK: true, Data: school})
}

func (h *Handler) UpdateSchool(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.CurrentUser(r.Context())
	if !ok {
		writeJSON(w, r, http.StatusUnauthorized, apiResponse{OK: false, Error: "unauthorized"})
		return
	}

	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		writeJSON(w, r, http.StatusBadRequest, apiResponse{OK: false, Error: "invalid id"})
		return
	}

	var req updateSchoolRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, r, http.StatusBadRequest, apiResponse{OK: false, Error: "invalid request body"})
		return
	}

	item, err := h.svc.UpdateSchool(r.Context(), user.ID, id, UpdateSchoolInput{
		Name:    req.Name,
		Code:    req.Code,
		Address: req.Address,
	})
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidInput):
			writeJSON(w, r, http.StatusBadRequest, apiResponse{OK: false, Error: "id and name are required"})
		case errors.Is(err, sql.ErrNoRows):
			writeJSON(w, r, http.StatusNotFound, apiResponse{OK: false, Error: "school not found"})
		default:
			writeJSON(w, r, http.StatusBadRequest, apiResponse{OK: false, Error: err.Error()})
		}
		return
	}
	writeJSON(w, r, http.StatusOK, apiResponse{OK: true, Data: item})
}

func (h *Handler) DeleteSchool(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.CurrentUser(r.Context())
	if !ok {
		writeJSON(w, r, http.StatusUnauthorized, apiResponse{OK: false, Error: "unauthorized"})
		return
	}

	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		writeJSON(w, r, http.StatusBadRequest, apiResponse{OK: false, Error: "invalid id"})
		return
	}

	err = h.svc.DeleteSchool(r.Context(), user.ID, id)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidInput):
			writeJSON(w, r, http.StatusBadRequest, apiResponse{OK: false, Error: "invalid id"})
		case errors.Is(err, sql.ErrNoRows):
			writeJSON(w, r, http.StatusNotFound, apiResponse{OK: false, Error: "school not found"})
		default:
			writeJSON(w, r, http.StatusBadRequest, apiResponse{OK: false, Error: err.Error()})
		}
		return
	}
	writeJSON(w, r, http.StatusOK, apiResponse{OK: true, Data: map[string]string{"status": "deleted"}})
}

func (h *Handler) CreateClass(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.CurrentUser(r.Context())
	if !ok {
		writeJSON(w, r, http.StatusUnauthorized, apiResponse{OK: false, Error: "unauthorized"})
		return
	}

	var req createClassRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, r, http.StatusBadRequest, apiResponse{OK: false, Error: "invalid request body"})
		return
	}

	class, err := h.svc.CreateClass(r.Context(), user.ID, CreateClassInput{
		SchoolID:   req.SchoolID,
		Name:       req.Name,
		GradeLevel: req.GradeLevel,
	})
	if err != nil {
		if errors.Is(err, ErrInvalidInput) {
			writeJSON(w, r, http.StatusBadRequest, apiResponse{OK: false, Error: "school_id, name, and grade_level are required"})
			return
		}
		writeJSON(w, r, http.StatusBadRequest, apiResponse{OK: false, Error: err.Error()})
		return
	}

	writeJSON(w, r, http.StatusCreated, apiResponse{OK: true, Data: class})
}

func (h *Handler) UpdateClass(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.CurrentUser(r.Context())
	if !ok {
		writeJSON(w, r, http.StatusUnauthorized, apiResponse{OK: false, Error: "unauthorized"})
		return
	}

	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		writeJSON(w, r, http.StatusBadRequest, apiResponse{OK: false, Error: "invalid id"})
		return
	}

	var req updateClassRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, r, http.StatusBadRequest, apiResponse{OK: false, Error: "invalid request body"})
		return
	}

	item, err := h.svc.UpdateClass(r.Context(), user.ID, id, UpdateClassInput{
		SchoolID:   req.SchoolID,
		Name:       req.Name,
		GradeLevel: req.GradeLevel,
	})
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidInput):
			writeJSON(w, r, http.StatusBadRequest, apiResponse{OK: false, Error: "id, school_id, name, and grade_level are required"})
		case errors.Is(err, sql.ErrNoRows):
			writeJSON(w, r, http.StatusNotFound, apiResponse{OK: false, Error: "class not found"})
		default:
			writeJSON(w, r, http.StatusBadRequest, apiResponse{OK: false, Error: err.Error()})
		}
		return
	}
	writeJSON(w, r, http.StatusOK, apiResponse{OK: true, Data: item})
}

func (h *Handler) DeleteClass(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.CurrentUser(r.Context())
	if !ok {
		writeJSON(w, r, http.StatusUnauthorized, apiResponse{OK: false, Error: "unauthorized"})
		return
	}

	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		writeJSON(w, r, http.StatusBadRequest, apiResponse{OK: false, Error: "invalid id"})
		return
	}

	err = h.svc.DeleteClass(r.Context(), user.ID, id)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidInput):
			writeJSON(w, r, http.StatusBadRequest, apiResponse{OK: false, Error: "invalid id"})
		case errors.Is(err, sql.ErrNoRows):
			writeJSON(w, r, http.StatusNotFound, apiResponse{OK: false, Error: "class not found"})
		default:
			writeJSON(w, r, http.StatusBadRequest, apiResponse{OK: false, Error: err.Error()})
		}
		return
	}
	writeJSON(w, r, http.StatusOK, apiResponse{OK: true, Data: map[string]string{"status": "deleted"}})
}

func (h *Handler) ImportStudentsCSV(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.CurrentUser(r.Context())
	if !ok {
		writeJSON(w, r, http.StatusUnauthorized, apiResponse{OK: false, Error: "unauthorized"})
		return
	}

	if err := r.ParseMultipartForm(16 << 20); err != nil {
		writeJSON(w, r, http.StatusBadRequest, apiResponse{OK: false, Error: "invalid multipart form"})
		return
	}

	file, hdr, err := r.FormFile("file")
	if err != nil {
		writeJSON(w, r, http.StatusBadRequest, apiResponse{OK: false, Error: "file field is required"})
		return
	}
	defer file.Close()

	report, err := h.svc.ImportStudentsCSV(r.Context(), user.ID, file)
	if err != nil {
		writeJSON(w, r, http.StatusBadRequest, apiResponse{OK: false, Error: err.Error()})
		return
	}

	writeJSON(w, r, http.StatusOK, apiResponse{OK: true, Data: map[string]any{
		"filename": hdr.Filename,
		"report":   report,
		"summary":  "import completed",
		"ok_rows":  strconv.Itoa(report.SuccessRows),
	}})
}

func writeJSON(w http.ResponseWriter, r *http.Request, code int, payload apiResponse) {
	apiresp.WriteLegacy(w, r, code, payload.OK, payload.Data, payload.Error)
}
