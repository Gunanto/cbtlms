package report

import "net/http"

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Summary(w http.ResponseWriter, r *http.Request) {
	_ = h
	_ = r
	w.WriteHeader(http.StatusNotImplemented)
	_, _ = w.Write([]byte("report handler not implemented"))
}
