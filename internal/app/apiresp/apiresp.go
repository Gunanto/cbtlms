package apiresp

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
)

type ErrorPayload struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type Meta struct {
	RequestID string `json:"request_id,omitempty"`
}

type Envelope struct {
	OK    bool          `json:"ok"`
	Data  interface{}   `json:"data,omitempty"`
	Error *ErrorPayload `json:"error,omitempty"`
	Meta  Meta          `json:"meta"`
}

func WriteOK(w http.ResponseWriter, r *http.Request, status int, data interface{}) {
	WriteLegacy(w, r, status, true, data, "")
}

func WriteError(w http.ResponseWriter, r *http.Request, status int, msg string) {
	WriteLegacy(w, r, status, false, nil, msg)
}

func WriteLegacy(w http.ResponseWriter, r *http.Request, status int, ok bool, data interface{}, errMsg string) {
	res := Envelope{
		OK: ok,
		Meta: Meta{
			RequestID: middleware.GetReqID(r.Context()),
		},
	}
	if ok {
		res.Data = data
	} else {
		if errMsg == "" {
			errMsg = http.StatusText(status)
		}
		res.Error = &ErrorPayload{
			Code:    codeFromStatus(status),
			Message: errMsg,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(res)
}

func codeFromStatus(status int) string {
	switch status {
	case http.StatusBadRequest:
		return "invalid_request"
	case http.StatusUnauthorized:
		return "unauthorized"
	case http.StatusForbidden:
		return "forbidden"
	case http.StatusNotFound:
		return "not_found"
	case http.StatusConflict:
		return "conflict"
	case http.StatusUnprocessableEntity:
		return "unprocessable_entity"
	case http.StatusTooManyRequests:
		return "rate_limited"
	case http.StatusInternalServerError:
		return "internal_error"
	default:
		if status >= 200 && status < 300 {
			return ""
		}
		return "error"
	}
}
