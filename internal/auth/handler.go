package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"cbtlms/internal/app/apiresp"

	"github.com/go-chi/chi/v5"
)

type contextKey string

const userContextKey contextKey = "auth_user"

const sessionCookieName = "cbtlms_session"
const csrfCookieName = "cbtlms_csrf"
const csrfHeaderName = "X-CSRF-Token"

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

type adminCreateUserRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
	FullName string `json:"full_name"`
	Role     string `json:"role"`
	SchoolID *int64 `json:"school_id"`
	ClassID  *int64 `json:"class_id"`
}

type adminUpdateUserRequest struct {
	FullName string `json:"full_name"`
	Email    string `json:"email"`
	Password string `json:"password"`
	Role     string `json:"role"`
	SchoolID *int64 `json:"school_id"`
	ClassID  *int64 `json:"class_id"`
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
		writeJSON(w, r, http.StatusBadRequest, apiResponse{OK: false, Error: "invalid request body"})
		return
	}

	user, err := h.svc.AuthenticatePassword(r.Context(), req.Identifier, req.Password)
	if err != nil {
		switch {
		case errors.Is(err, ErrRateLimited):
			writeJSON(w, r, http.StatusTooManyRequests, apiResponse{OK: false, Error: "too many attempts"})
		case errors.Is(err, ErrInvalidCredentials):
			writeJSON(w, r, http.StatusUnauthorized, apiResponse{OK: false, Error: "invalid credentials"})
		case errors.Is(err, ErrForbidden):
			writeJSON(w, r, http.StatusForbidden, apiResponse{OK: false, Error: "account is not active"})
		default:
			writeJSON(w, r, http.StatusInternalServerError, apiResponse{OK: false, Error: "internal error"})
		}
		return
	}

	if err := h.establishSession(w, r, user); err != nil {
		writeJSON(w, r, http.StatusInternalServerError, apiResponse{OK: false, Error: "cannot create session"})
		return
	}
	writeJSON(w, r, http.StatusOK, apiResponse{OK: true, Data: user})
}

func (h *Handler) RequestOTP(w http.ResponseWriter, r *http.Request) {
	var req otpRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, r, http.StatusBadRequest, apiResponse{OK: false, Error: "invalid request body"})
		return
	}

	err := h.svc.RequestOTP(r.Context(), req.Email)
	if err != nil {
		if errors.Is(err, ErrRateLimited) {
			writeJSON(w, r, http.StatusTooManyRequests, apiResponse{OK: false, Error: "otp requested too frequently"})
			return
		}
		writeJSON(w, r, http.StatusInternalServerError, apiResponse{OK: false, Error: "internal error"})
		return
	}
	writeJSON(w, r, http.StatusOK, apiResponse{OK: true, Data: map[string]string{"status": "if eligible, otp has been sent"}})
}

func (h *Handler) VerifyOTP(w http.ResponseWriter, r *http.Request) {
	var req otpVerifyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, r, http.StatusBadRequest, apiResponse{OK: false, Error: "invalid request body"})
		return
	}

	user, err := h.svc.VerifyOTP(r.Context(), req.Email, req.Code)
	if err != nil {
		switch {
		case errors.Is(err, ErrRateLimited):
			writeJSON(w, r, http.StatusTooManyRequests, apiResponse{OK: false, Error: "too many otp attempts"})
		case errors.Is(err, ErrInvalidOTP):
			writeJSON(w, r, http.StatusUnauthorized, apiResponse{OK: false, Error: "invalid otp"})
		case errors.Is(err, ErrOTPExpired):
			writeJSON(w, r, http.StatusUnauthorized, apiResponse{OK: false, Error: "otp expired"})
		case errors.Is(err, ErrForbidden):
			writeJSON(w, r, http.StatusForbidden, apiResponse{OK: false, Error: "account is not active"})
		default:
			writeJSON(w, r, http.StatusInternalServerError, apiResponse{OK: false, Error: "internal error"})
		}
		return
	}

	if err := h.establishSession(w, r, user); err != nil {
		writeJSON(w, r, http.StatusInternalServerError, apiResponse{OK: false, Error: "cannot create session"})
		return
	}
	writeJSON(w, r, http.StatusOK, apiResponse{OK: true, Data: user})
}

func (h *Handler) BootstrapInit(w http.ResponseWriter, r *http.Request) {
	var req bootstrapInitRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, r, http.StatusBadRequest, apiResponse{OK: false, Error: "invalid request body"})
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
			writeJSON(w, r, http.StatusForbidden, apiResponse{OK: false, Error: "bootstrap denied"})
		default:
			writeJSON(w, r, http.StatusBadRequest, apiResponse{OK: false, Error: err.Error()})
		}
		return
	}

	writeJSON(w, r, http.StatusOK, apiResponse{OK: true, Data: map[string]string{"status": "bootstrap updated"}})
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
	http.SetCookie(w, &http.Cookie{
		Name:     csrfCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: false,
		Secure:   false,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})

	writeJSON(w, r, http.StatusOK, apiResponse{OK: true, Data: map[string]string{"status": "logged_out"}})
}

func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	user, ok := CurrentUser(r.Context())
	if !ok {
		writeJSON(w, r, http.StatusUnauthorized, apiResponse{OK: false, Error: "unauthorized"})
		return
	}
	writeJSON(w, r, http.StatusOK, apiResponse{OK: true, Data: user})
}

func (h *Handler) CreateRegistration(w http.ResponseWriter, r *http.Request) {
	var req createRegistrationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, r, http.StatusBadRequest, apiResponse{OK: false, Error: "invalid request body"})
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
		writeJSON(w, r, http.StatusBadRequest, apiResponse{OK: false, Error: err.Error()})
		return
	}

	writeJSON(w, r, http.StatusCreated, apiResponse{OK: true, Data: map[string]interface{}{"registration_id": id, "status": "pending"}})
}

func (h *Handler) ListPendingRegistrations(w http.ResponseWriter, r *http.Request) {
	status := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("status")))
	limit := 50
	offset := 0
	if v := strings.TrimSpace(r.URL.Query().Get("limit")); v != "" {
		n, err := strconv.Atoi(v)
		if err == nil {
			limit = n
		}
	}
	if v := strings.TrimSpace(r.URL.Query().Get("offset")); v != "" {
		n, err := strconv.Atoi(v)
		if err == nil {
			offset = n
		}
	}

	items, err := h.svc.ListRegistrations(r.Context(), status, limit, offset)
	if err != nil {
		if strings.Contains(err.Error(), "invalid status filter") {
			writeJSON(w, r, http.StatusBadRequest, apiResponse{OK: false, Error: err.Error()})
			return
		}
		writeJSON(w, r, http.StatusInternalServerError, apiResponse{OK: false, Error: "internal error"})
		return
	}

	writeJSON(w, r, http.StatusOK, apiResponse{OK: true, Data: map[string]any{
		"items":  items,
		"limit":  limit,
		"offset": offset,
		"status": status,
	}})
}

func (h *Handler) ListUsers(w http.ResponseWriter, r *http.Request) {
	role := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("role")))
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	limit := 50
	offset := 0
	if v := strings.TrimSpace(r.URL.Query().Get("limit")); v != "" {
		n, err := strconv.Atoi(v)
		if err == nil {
			limit = n
		}
	}
	if v := strings.TrimSpace(r.URL.Query().Get("offset")); v != "" {
		n, err := strconv.Atoi(v)
		if err == nil {
			offset = n
		}
	}

	items, err := h.svc.ListUsers(r.Context(), role, q, limit, offset)
	if err != nil {
		if strings.Contains(err.Error(), "invalid role filter") {
			writeJSON(w, r, http.StatusBadRequest, apiResponse{OK: false, Error: err.Error()})
			return
		}
		writeJSON(w, r, http.StatusInternalServerError, apiResponse{OK: false, Error: "internal error"})
		return
	}

	writeJSON(w, r, http.StatusOK, apiResponse{OK: true, Data: map[string]any{
		"items":  items,
		"limit":  limit,
		"offset": offset,
		"role":   role,
		"q":      q,
	}})
}

func (h *Handler) DashboardStats(w http.ResponseWriter, r *http.Request) {
	out, err := h.svc.AdminDashboardStats(r.Context())
	if err != nil {
		writeJSON(w, r, http.StatusInternalServerError, apiResponse{OK: false, Error: "internal error"})
		return
	}
	writeJSON(w, r, http.StatusOK, apiResponse{OK: true, Data: out})
}

func (h *Handler) CreateUser(w http.ResponseWriter, r *http.Request) {
	admin, ok := CurrentUser(r.Context())
	if !ok {
		writeJSON(w, r, http.StatusUnauthorized, apiResponse{OK: false, Error: "unauthorized"})
		return
	}

	var req adminCreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, r, http.StatusBadRequest, apiResponse{OK: false, Error: "invalid request body"})
		return
	}

	out, err := h.svc.CreateUserByAdmin(r.Context(), admin.ID, AdminCreateUserInput{
		Username: req.Username,
		Email:    req.Email,
		Password: req.Password,
		FullName: req.FullName,
		Role:     req.Role,
		SchoolID: req.SchoolID,
		ClassID:  req.ClassID,
	})
	if err != nil {
		writeJSON(w, r, http.StatusBadRequest, apiResponse{OK: false, Error: err.Error()})
		return
	}
	writeJSON(w, r, http.StatusCreated, apiResponse{OK: true, Data: out})
}

func (h *Handler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	admin, ok := CurrentUser(r.Context())
	if !ok {
		writeJSON(w, r, http.StatusUnauthorized, apiResponse{OK: false, Error: "unauthorized"})
		return
	}

	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		writeJSON(w, r, http.StatusBadRequest, apiResponse{OK: false, Error: "invalid user id"})
		return
	}

	var req adminUpdateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, r, http.StatusBadRequest, apiResponse{OK: false, Error: "invalid request body"})
		return
	}

	out, err := h.svc.UpdateUserByAdmin(r.Context(), admin.ID, id, AdminUpdateUserInput{
		FullName: req.FullName,
		Email:    req.Email,
		Password: req.Password,
		Role:     req.Role,
		SchoolID: req.SchoolID,
		ClassID:  req.ClassID,
	})
	if err != nil {
		switch {
		case errors.Is(err, ErrUserNotFound):
			writeJSON(w, r, http.StatusNotFound, apiResponse{OK: false, Error: err.Error()})
		default:
			writeJSON(w, r, http.StatusBadRequest, apiResponse{OK: false, Error: err.Error()})
		}
		return
	}
	writeJSON(w, r, http.StatusOK, apiResponse{OK: true, Data: out})
}

func (h *Handler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	admin, ok := CurrentUser(r.Context())
	if !ok {
		writeJSON(w, r, http.StatusUnauthorized, apiResponse{OK: false, Error: "unauthorized"})
		return
	}

	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		writeJSON(w, r, http.StatusBadRequest, apiResponse{OK: false, Error: "invalid user id"})
		return
	}
	if id == admin.ID {
		writeJSON(w, r, http.StatusBadRequest, apiResponse{OK: false, Error: "cannot deactivate your own account"})
		return
	}

	err = h.svc.DeactivateUserByAdmin(r.Context(), admin.ID, id)
	if err != nil {
		switch {
		case errors.Is(err, ErrUserNotFound):
			writeJSON(w, r, http.StatusNotFound, apiResponse{OK: false, Error: err.Error()})
		default:
			writeJSON(w, r, http.StatusBadRequest, apiResponse{OK: false, Error: err.Error()})
		}
		return
	}
	writeJSON(w, r, http.StatusOK, apiResponse{OK: true, Data: map[string]string{"status": "deactivated"}})
}

func (h *Handler) ExportUsersExcel(w http.ResponseWriter, r *http.Request) {
	role := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("role")))
	q := strings.TrimSpace(r.URL.Query().Get("q"))

	content, err := h.svc.ExportUsersExcel(r.Context(), role, q)
	if err != nil {
		if strings.Contains(err.Error(), "invalid role filter") {
			writeJSON(w, r, http.StatusBadRequest, apiResponse{OK: false, Error: err.Error()})
			return
		}
		writeJSON(w, r, http.StatusInternalServerError, apiResponse{OK: false, Error: "internal error"})
		return
	}

	filename := fmt.Sprintf("daftar_pengguna_%s.xlsx", time.Now().Format("20060102_150405"))
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(content)
}

func (h *Handler) ImportUsersExcel(w http.ResponseWriter, r *http.Request) {
	admin, ok := CurrentUser(r.Context())
	if !ok {
		writeJSON(w, r, http.StatusUnauthorized, apiResponse{OK: false, Error: "unauthorized"})
		return
	}

	if err := r.ParseMultipartForm(32 << 20); err != nil {
		writeJSON(w, r, http.StatusBadRequest, apiResponse{OK: false, Error: "invalid multipart form"})
		return
	}

	file, hdr, err := r.FormFile("file")
	if err != nil {
		writeJSON(w, r, http.StatusBadRequest, apiResponse{OK: false, Error: "file field is required"})
		return
	}
	defer file.Close()

	report, err := h.svc.ImportUsersExcel(r.Context(), admin.ID, file)
	if err != nil {
		writeJSON(w, r, http.StatusBadRequest, apiResponse{OK: false, Error: err.Error()})
		return
	}

	writeJSON(w, r, http.StatusOK, apiResponse{OK: true, Data: map[string]any{
		"filename": hdr.Filename,
		"report":   report,
		"summary":  "import completed",
	}})
}

func (h *Handler) ApproveRegistration(w http.ResponseWriter, r *http.Request) {
	admin, ok := CurrentUser(r.Context())
	if !ok {
		writeJSON(w, r, http.StatusUnauthorized, apiResponse{OK: false, Error: "unauthorized"})
		return
	}

	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		writeJSON(w, r, http.StatusBadRequest, apiResponse{OK: false, Error: "invalid registration id"})
		return
	}

	userID, err := h.svc.ApproveRegistration(r.Context(), id, admin.ID)
	if err != nil {
		switch {
		case errors.Is(err, ErrRegistrationNotFound):
			writeJSON(w, r, http.StatusNotFound, apiResponse{OK: false, Error: err.Error()})
		case errors.Is(err, ErrRegistrationState):
			writeJSON(w, r, http.StatusConflict, apiResponse{OK: false, Error: err.Error()})
		default:
			writeJSON(w, r, http.StatusInternalServerError, apiResponse{OK: false, Error: "internal error"})
		}
		return
	}

	writeJSON(w, r, http.StatusOK, apiResponse{OK: true, Data: map[string]interface{}{"status": "approved", "user_id": userID}})
}

func (h *Handler) RejectRegistration(w http.ResponseWriter, r *http.Request) {
	admin, ok := CurrentUser(r.Context())
	if !ok {
		writeJSON(w, r, http.StatusUnauthorized, apiResponse{OK: false, Error: "unauthorized"})
		return
	}

	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		writeJSON(w, r, http.StatusBadRequest, apiResponse{OK: false, Error: "invalid registration id"})
		return
	}

	var req rejectRegistrationRequest
	_ = json.NewDecoder(r.Body).Decode(&req)

	err = h.svc.RejectRegistration(r.Context(), id, admin.ID, req.Note)
	if err != nil {
		switch {
		case errors.Is(err, ErrRegistrationState):
			writeJSON(w, r, http.StatusConflict, apiResponse{OK: false, Error: err.Error()})
		default:
			writeJSON(w, r, http.StatusInternalServerError, apiResponse{OK: false, Error: "internal error"})
		}
		return
	}

	writeJSON(w, r, http.StatusOK, apiResponse{OK: true, Data: map[string]string{"status": "rejected"}})
}

func (h *Handler) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := readSessionToken(r)
		user, err := h.svc.GetSessionUser(r.Context(), token)
		if err != nil {
			writeJSON(w, r, http.StatusUnauthorized, apiResponse{OK: false, Error: "unauthorized"})
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
				writeJSON(w, r, http.StatusUnauthorized, apiResponse{OK: false, Error: "unauthorized"})
				return
			}
			if _, exists := allowed[user.Role]; !exists {
				writeJSON(w, r, http.StatusForbidden, apiResponse{OK: false, Error: "forbidden"})
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
	csrfToken, err := generateCSRFToken()
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
	http.SetCookie(w, &http.Cookie{
		Name:     csrfCookieName,
		Value:    csrfToken,
		Path:     "/",
		Expires:  expiresAt,
		HttpOnly: false,
		Secure:   false,
		SameSite: http.SameSiteLaxMode,
	})
	w.Header().Set(csrfHeaderName, csrfToken)
	return nil
}

func generateCSRFToken() (string, error) {
	var b [32]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b[:]), nil
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

func writeJSON(w http.ResponseWriter, r *http.Request, code int, payload apiResponse) {
	apiresp.WriteLegacy(w, r, code, payload.OK, payload.Data, payload.Error)
}
