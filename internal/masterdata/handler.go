package masterdata

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"cbtlms/internal/app/apiresp"
	"cbtlms/internal/auth"
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

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
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
