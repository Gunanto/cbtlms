package assistant

import (
	"encoding/json"
	"net/http"
	"strings"

	"cbtlms/internal/app/apiresp"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

type replyRequest struct {
	Query string `json:"query"`
}

func (h *Handler) Reply(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		apiresp.WriteError(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req replyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apiresp.WriteError(w, r, http.StatusBadRequest, "payload tidak valid")
		return
	}
	if strings.TrimSpace(req.Query) == "" {
		apiresp.WriteError(w, r, http.StatusBadRequest, "query wajib diisi")
		return
	}

	result, err := h.svc.Generate(r.Context(), req.Query)
	if err != nil {
		apiresp.WriteError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	apiresp.WriteOK(w, r, http.StatusOK, map[string]any{
		"reply":  result.Reply,
		"source": result.Source,
	})
}
