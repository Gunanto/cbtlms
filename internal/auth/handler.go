package auth

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
)

type contextKey string

const userContextKey contextKey = "auth_user"

const sessionCookieName = "cbtlms_session"

type Handler struct {
	svc *Service
}

type apiResponse struct {
	OK    bool        `json:"ok"`
	Data  interface{} `json:"data,omitempty"`
	Error string      `json:"error,omitempty"`
}

type loginPasswordRequest struct {
	Identifier string `json:"identifier"`
	Password   string `json:"password"`
}

type otpRequest struct {
	Email string `json:"email"`
}

type otpVerifyRequest struct {
	Email string `json:"email"`
	Code  string `json:"code"`
}

type createRegistrationRequest struct {
	RoleRequested   string          `json:"role_requested"`
	Email           string          `json:"email"`
	Password        string          `json:"password"`
	FullName        string          `json:"full_name"`
	Phone           string          `json:"phone"`
	InstitutionName string          `json:"institution_name"`
	FormPayload     json.RawMessage `json:"form_payload"`
}

type rejectRegistrationRequest struct {
	Note string `json:"note"`
}

type bootstrapInitRequest struct {
	Token           string `json:"token"`
	AdminUsername   string `json:"admin_username"`
	AdminEmail      string `json:"admin_email"`
	AdminPassword   string `json:"admin_password"`
	ProktorUsername string `json:"proktor_username"`
	ProktorEmail    string `json:"proktor_email"`
	ProktorPassword string `json:"proktor_password"`
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) LoginPassword(w http.ResponseWriter, r *http.Request) {
	var req loginPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, apiResponse{OK: false, Error: "invalid request body"})
		return
	}

	user, err := h.svc.AuthenticatePassword(r.Context(), req.Identifier, req.Password)
	if err != nil {
		switch {
		case errors.Is(err, ErrRateLimited):
			writeJSON(w, http.StatusTooManyRequests, apiResponse{OK: false, Error: "too many attempts"})
		case errors.Is(err, ErrInvalidCredentials):
			writeJSON(w, http.StatusUnauthorized, apiResponse{OK: false, Error: "invalid credentials"})
		case errors.Is(err, ErrForbidden):
			writeJSON(w, http.StatusForbidden, apiResponse{OK: false, Error: "account is not active"})
		default:
			writeJSON(w, http.StatusInternalServerError, apiResponse{OK: false, Error: "internal error"})
		}
		return
	}

	if err := h.establishSession(w, r, user); err != nil {
		writeJSON(w, http.StatusInternalServerError, apiResponse{OK: false, Error: "cannot create session"})
		return
	}
	writeJSON(w, http.StatusOK, apiResponse{OK: true, Data: user})
}

func (h *Handler) RequestOTP(w http.ResponseWriter, r *http.Request) {
	var req otpRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, apiResponse{OK: false, Error: "invalid request body"})
		return
	}

	err := h.svc.RequestOTP(r.Context(), req.Email)
	if err != nil {
		if errors.Is(err, ErrRateLimited) {
			writeJSON(w, http.StatusTooManyRequests, apiResponse{OK: false, Error: "otp requested too frequently"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, apiResponse{OK: false, Error: "internal error"})
		return
	}
	writeJSON(w, http.StatusOK, apiResponse{OK: true, Data: map[string]string{"status": "if eligible, otp has been sent"}})
}

func (h *Handler) VerifyOTP(w http.ResponseWriter, r *http.Request) {
	var req otpVerifyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, apiResponse{OK: false, Error: "invalid request body"})
		return
	}

	user, err := h.svc.VerifyOTP(r.Context(), req.Email, req.Code)
	if err != nil {
		switch {
		case errors.Is(err, ErrRateLimited):
			writeJSON(w, http.StatusTooManyRequests, apiResponse{OK: false, Error: "too many otp attempts"})
		case errors.Is(err, ErrInvalidOTP):
			writeJSON(w, http.StatusUnauthorized, apiResponse{OK: false, Error: "invalid otp"})
		case errors.Is(err, ErrOTPExpired):
			writeJSON(w, http.StatusUnauthorized, apiResponse{OK: false, Error: "otp expired"})
		case errors.Is(err, ErrForbidden):
			writeJSON(w, http.StatusForbidden, apiResponse{OK: false, Error: "account is not active"})
		default:
			writeJSON(w, http.StatusInternalServerError, apiResponse{OK: false, Error: "internal error"})
		}
		return
	}

	if err := h.establishSession(w, r, user); err != nil {
		writeJSON(w, http.StatusInternalServerError, apiResponse{OK: false, Error: "cannot create session"})
		return
	}
	writeJSON(w, http.StatusOK, apiResponse{OK: true, Data: user})
}

func (h *Handler) BootstrapInit(w http.ResponseWriter, r *http.Request) {
	var req bootstrapInitRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, apiResponse{OK: false, Error: "invalid request body"})
		return
	}

	err := h.svc.BootstrapAccounts(r.Context(), BootstrapInput{
		Token:           req.Token,
		AdminUsername:   req.AdminUsername,
		AdminEmail:      req.AdminEmail,
		AdminPassword:   req.AdminPassword,
		ProktorUsername: req.ProktorUsername,
		ProktorEmail:    req.ProktorEmail,
		ProktorPassword: req.ProktorPassword,
	})
	if err != nil {
		switch {
		case errors.Is(err, ErrBootstrapDenied):
			writeJSON(w, http.StatusForbidden, apiResponse{OK: false, Error: "bootstrap denied"})
		default:
			writeJSON(w, http.StatusBadRequest, apiResponse{OK: false, Error: err.Error()})
		}
		return
	}

	writeJSON(w, http.StatusOK, apiResponse{OK: true, Data: map[string]string{"status": "bootstrap updated"}})
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	token := readSessionToken(r)
	_ = h.svc.RevokeSession(r.Context(), token)

	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   false,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})

	writeJSON(w, http.StatusOK, apiResponse{OK: true, Data: map[string]string{"status": "logged_out"}})
}

func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	user, ok := CurrentUser(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, apiResponse{OK: false, Error: "unauthorized"})
		return
	}
	writeJSON(w, http.StatusOK, apiResponse{OK: true, Data: user})
}

func (h *Handler) CreateRegistration(w http.ResponseWriter, r *http.Request) {
	var req createRegistrationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, apiResponse{OK: false, Error: "invalid request body"})
		return
	}

	formPayload := strings.TrimSpace(string(req.FormPayload))
	if formPayload == "" {
		formPayload = "{}"
	}

	id, err := h.svc.CreateRegistration(r.Context(), RegistrationInput{
		RoleRequested:   req.RoleRequested,
		Email:           req.Email,
		Password:        req.Password,
		FullName:        req.FullName,
		Phone:           req.Phone,
		InstitutionName: req.InstitutionName,
		FormPayload:     formPayload,
	})
	if err != nil {
		writeJSON(w, http.StatusBadRequest, apiResponse{OK: false, Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusCreated, apiResponse{OK: true, Data: map[string]interface{}{"registration_id": id, "status": "pending"}})
}

func (h *Handler) ListPendingRegistrations(w http.ResponseWriter, r *http.Request) {
	limit := 50
	if v := strings.TrimSpace(r.URL.Query().Get("limit")); v != "" {
		n, err := strconv.Atoi(v)
		if err == nil {
			limit = n
		}
	}

	items, err := h.svc.ListRegistrationPending(r.Context(), limit)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiResponse{OK: false, Error: "internal error"})
		return
	}

	writeJSON(w, http.StatusOK, apiResponse{OK: true, Data: items})
}

func (h *Handler) ApproveRegistration(w http.ResponseWriter, r *http.Request) {
	admin, ok := CurrentUser(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, apiResponse{OK: false, Error: "unauthorized"})
		return
	}

	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		writeJSON(w, http.StatusBadRequest, apiResponse{OK: false, Error: "invalid registration id"})
		return
	}

	userID, err := h.svc.ApproveRegistration(r.Context(), id, admin.ID)
	if err != nil {
		switch {
		case errors.Is(err, ErrRegistrationNotFound):
			writeJSON(w, http.StatusNotFound, apiResponse{OK: false, Error: err.Error()})
		case errors.Is(err, ErrRegistrationState):
			writeJSON(w, http.StatusConflict, apiResponse{OK: false, Error: err.Error()})
		default:
			writeJSON(w, http.StatusInternalServerError, apiResponse{OK: false, Error: "internal error"})
		}
		return
	}

	writeJSON(w, http.StatusOK, apiResponse{OK: true, Data: map[string]interface{}{"status": "approved", "user_id": userID}})
}

func (h *Handler) RejectRegistration(w http.ResponseWriter, r *http.Request) {
	admin, ok := CurrentUser(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, apiResponse{OK: false, Error: "unauthorized"})
		return
	}

	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		writeJSON(w, http.StatusBadRequest, apiResponse{OK: false, Error: "invalid registration id"})
		return
	}

	var req rejectRegistrationRequest
	_ = json.NewDecoder(r.Body).Decode(&req)

	err = h.svc.RejectRegistration(r.Context(), id, admin.ID, req.Note)
	if err != nil {
		switch {
		case errors.Is(err, ErrRegistrationState):
			writeJSON(w, http.StatusConflict, apiResponse{OK: false, Error: err.Error()})
		default:
			writeJSON(w, http.StatusInternalServerError, apiResponse{OK: false, Error: "internal error"})
		}
		return
	}

	writeJSON(w, http.StatusOK, apiResponse{OK: true, Data: map[string]string{"status": "rejected"}})
}

func (h *Handler) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := readSessionToken(r)
		user, err := h.svc.GetSessionUser(r.Context(), token)
		if err != nil {
			writeJSON(w, http.StatusUnauthorized, apiResponse{OK: false, Error: "unauthorized"})
			return
		}

		ctx := context.WithValue(r.Context(), userContextKey, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (h *Handler) RequireRoles(roles ...string) func(http.Handler) http.Handler {
	allowed := make(map[string]struct{}, len(roles))
	for _, role := range roles {
		allowed[role] = struct{}{}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, ok := CurrentUser(r.Context())
			if !ok {
				writeJSON(w, http.StatusUnauthorized, apiResponse{OK: false, Error: "unauthorized"})
				return
			}
			if _, exists := allowed[user.Role]; !exists {
				writeJSON(w, http.StatusForbidden, apiResponse{OK: false, Error: "forbidden"})
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func CurrentUser(ctx context.Context) (*User, bool) {
	v := ctx.Value(userContextKey)
	if v == nil {
		return nil, false
	}
	u, ok := v.(*User)
	return u, ok
}

// ContextWithUser injects an authenticated user into context.
// Useful for tests and internal handlers.
func ContextWithUser(ctx context.Context, user *User) context.Context {
	return context.WithValue(ctx, userContextKey, user)
}

func (h *Handler) establishSession(w http.ResponseWriter, r *http.Request, user *User) error {
	token, expiresAt, err := h.svc.CreateSession(r.Context(), user.ID, readIP(r), r.UserAgent())
	if err != nil {
		return err
	}

	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    token,
		Path:     "/",
		Expires:  expiresAt,
		HttpOnly: true,
		Secure:   false,
		SameSite: http.SameSiteLaxMode,
	})
	return nil
}

func readSessionToken(r *http.Request) string {
	c, err := r.Cookie(sessionCookieName)
	if err != nil {
		return ""
	}
	return c.Value
}

func readIP(r *http.Request) string {
	xff := strings.TrimSpace(r.Header.Get("X-Forwarded-For"))
	if xff != "" {
		parts := strings.Split(xff, ",")
		if len(parts) > 0 {
			return strings.TrimSpace(parts[0])
		}
	}
	return strings.TrimSpace(r.RemoteAddr)
}

func writeJSON(w http.ResponseWriter, code int, payload apiResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(payload)
}
