package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"math/big"
	"net/mail"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidCredentials   = errors.New("invalid credentials")
	ErrUnauthorized         = errors.New("unauthorized")
	ErrForbidden            = errors.New("forbidden")
	ErrInvalidOTP           = errors.New("invalid otp")
	ErrOTPExpired           = errors.New("otp expired")
	ErrRegistrationNotFound = errors.New("registration not found")
	ErrRegistrationState    = errors.New("registration is not pending")
	ErrRateLimited          = errors.New("too many requests")
	ErrBootstrapDenied      = errors.New("bootstrap denied")
	ErrUserNotFound         = errors.New("user not found")
)

type Service struct {
	db                *sql.DB
	sessionTTL        time.Duration
	otpTTL            time.Duration
	otpResendCooldown time.Duration
	bcryptCost        int
	loginMaxFailures  int
	loginLockDuration time.Duration
	otpMaxFailures    int
	otpLockDuration   time.Duration
	mailer            OTPMailer
	bootstrapToken    string
}

type ServiceConfig struct {
	SessionTTL        time.Duration
	OTPTTL            time.Duration
	OTPResendCooldown time.Duration
	BcryptCost        int
	LoginMaxFailures  int
	LoginLockDuration time.Duration
	OTPMaxFailures    int
	OTPLockDuration   time.Duration
	BootstrapToken    string
	Mailer            OTPMailer
}

type User struct {
	ID            int64      `json:"id"`
	Username      string     `json:"username"`
	Email         *string    `json:"email,omitempty"`
	FullName      string     `json:"full_name"`
	Role          string     `json:"role"`
	AccountStatus string     `json:"account_status"`
	ApprovedAt    *time.Time `json:"approved_at,omitempty"`
}

type RegistrationInput struct {
	RoleRequested   string
	Email           string
	Password        string
	FullName        string
	Phone           string
	InstitutionName string
	FormPayload     string
}

type RegistrationRecord struct {
	ID            int64      `json:"id"`
	RoleRequested string     `json:"role_requested"`
	Email         string     `json:"email"`
	FullName      string     `json:"full_name"`
	Phone         *string    `json:"phone,omitempty"`
	Institution   *string    `json:"institution_name,omitempty"`
	Status        string     `json:"status"`
	ReviewNote    *string    `json:"review_note,omitempty"`
	ReviewedBy    *int64     `json:"reviewed_by,omitempty"`
	ReviewedAt    *time.Time `json:"reviewed_at,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
}

type AdminUserRecord struct {
	ID            int64      `json:"id"`
	Username      string     `json:"username"`
	Email         *string    `json:"email,omitempty"`
	FullName      string     `json:"full_name"`
	Role          string     `json:"role"`
	SchoolID      *int64     `json:"school_id,omitempty"`
	SchoolName    *string    `json:"school_name,omitempty"`
	ClassID       *int64     `json:"class_id,omitempty"`
	ClassName     *string    `json:"class_name,omitempty"`
	IsActive      bool       `json:"is_active"`
	AccountStatus string     `json:"account_status"`
	ApprovedAt    *time.Time `json:"approved_at,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
}

type AdminDashboardStats struct {
	AdminCount   int64 `json:"admin_count"`
	ProktorCount int64 `json:"proktor_count"`
	GuruCount    int64 `json:"guru_count"`
	SiswaCount   int64 `json:"siswa_count"`
	SchoolCount  int64 `json:"school_count"`
}

type AdminCreateUserInput struct {
	Username string
	Email    string
	Password string
	FullName string
	Role     string
	SchoolID *int64
	ClassID  *int64
}

type AdminUpdateUserInput struct {
	FullName string
	Email    string
	Role     string
	Password string
	SchoolID *int64
	ClassID  *int64
}

type BootstrapInput struct {
	Token           string
	AdminUsername   string
	AdminEmail      string
	AdminPassword   string
	ProktorUsername string
	ProktorEmail    string
	ProktorPassword string
}

func NewService(db *sql.DB, cfg ServiceConfig) *Service {
	if cfg.SessionTTL <= 0 {
		cfg.SessionTTL = 24 * time.Hour
	}
	if cfg.OTPTTL <= 0 {
		cfg.OTPTTL = 5 * time.Minute
	}
	if cfg.OTPResendCooldown <= 0 {
		cfg.OTPResendCooldown = 60 * time.Second
	}
	if cfg.BcryptCost <= 0 {
		cfg.BcryptCost = bcrypt.DefaultCost
	}
	if cfg.LoginMaxFailures <= 0 {
		cfg.LoginMaxFailures = 5
	}
	if cfg.LoginLockDuration <= 0 {
		cfg.LoginLockDuration = 15 * time.Minute
	}
	if cfg.OTPMaxFailures <= 0 {
		cfg.OTPMaxFailures = 5
	}
	if cfg.OTPLockDuration <= 0 {
		cfg.OTPLockDuration = 10 * time.Minute
	}

	return &Service{
		db:                db,
		sessionTTL:        cfg.SessionTTL,
		otpTTL:            cfg.OTPTTL,
		otpResendCooldown: cfg.OTPResendCooldown,
		bcryptCost:        cfg.BcryptCost,
		loginMaxFailures:  cfg.LoginMaxFailures,
		loginLockDuration: cfg.LoginLockDuration,
		otpMaxFailures:    cfg.OTPMaxFailures,
		otpLockDuration:   cfg.OTPLockDuration,
		mailer:            cfg.Mailer,
		bootstrapToken:    strings.TrimSpace(cfg.BootstrapToken),
	}
}

func (s *Service) AuthenticatePassword(ctx context.Context, identifier, password string) (*User, error) {
	identifier = strings.TrimSpace(identifier)
	if identifier == "" || password == "" {
		return nil, ErrInvalidCredentials
	}

	guardKey := normalizeGuardKey(identifier)
	locked, _, err := s.isGuardLocked(ctx, "password_login", guardKey)
	if err != nil {
		return nil, fmt.Errorf("check login guard: %w", err)
	}
	if locked {
		return nil, ErrRateLimited
	}

	row := s.db.QueryRowContext(ctx, `
		SELECT id, username, email, full_name, role, account_status, approved_at, password_hash
		FROM users
		WHERE username = $1 OR email = $1
		LIMIT 1
	`, identifier)

	var u User
	var email sql.NullString
	var approvedAt sql.NullTime
	var passwordHash string
	if err := row.Scan(&u.ID, &u.Username, &email, &u.FullName, &u.Role, &u.AccountStatus, &approvedAt, &passwordHash); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			_ = s.registerFailure(ctx, "password_login", guardKey, s.loginMaxFailures, s.loginLockDuration)
			return nil, ErrInvalidCredentials
		}
		return nil, fmt.Errorf("query user: %w", err)
	}
	if email.Valid {
		u.Email = &email.String
	}
	if approvedAt.Valid {
		u.ApprovedAt = &approvedAt.Time
	}

	if u.AccountStatus != "active" {
		_ = s.registerFailure(ctx, "password_login", guardKey, s.loginMaxFailures, s.loginLockDuration)
		return nil, ErrForbidden
	}

	if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(password)); err != nil {
		_ = s.registerFailure(ctx, "password_login", guardKey, s.loginMaxFailures, s.loginLockDuration)
		return nil, ErrInvalidCredentials
	}

	_ = s.clearGuard(ctx, "password_login", guardKey)
	return &u, nil
}

func (s *Service) RequestOTP(ctx context.Context, email string) error {
	email = strings.ToLower(strings.TrimSpace(email))
	if _, err := mail.ParseAddress(email); err != nil {
		return nil
	}

	locked, _, err := s.isGuardLocked(ctx, "otp_request", normalizeGuardKey(email))
	if err != nil {
		return fmt.Errorf("check otp request guard: %w", err)
	}
	if locked {
		return ErrRateLimited
	}

	var userID int64
	err = s.db.QueryRowContext(ctx, `
		SELECT id
		FROM users
		WHERE email = $1
		  AND account_status = 'active'
		  AND email_verified_at IS NOT NULL
		LIMIT 1
	`, email).Scan(&userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		return fmt.Errorf("query otp user: %w", err)
	}

	var lastCreated time.Time
	err = s.db.QueryRowContext(ctx, `
		SELECT created_at
		FROM auth_otp_codes
		WHERE email = $1 AND purpose = 'login'
		ORDER BY created_at DESC
		LIMIT 1
	`, email).Scan(&lastCreated)
	if err == nil && time.Since(lastCreated) < s.otpResendCooldown {
		return ErrRateLimited
	}
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("query otp latest: %w", err)
	}

	otpCode, err := generateOTPCode(6)
	if err != nil {
		return fmt.Errorf("generate otp: %w", err)
	}
	otpHash := hashOTP(email, otpCode)
	expiresAt := time.Now().Add(s.otpTTL)

	_, err = s.db.ExecContext(ctx, `
		INSERT INTO auth_otp_codes (
			user_id, email, code_hash, purpose, expires_at, created_at
		) VALUES (
			$1, $2, $3, 'login', $4, now()
		)
	`, userID, email, otpHash, expiresAt)
	if err != nil {
		return fmt.Errorf("insert otp: %w", err)
	}

	if s.mailer != nil {
		if err := s.mailer.SendOTP(ctx, email, otpCode); err != nil {
			log.Printf("smtp otp send failed email=%s err=%v", email, err)
			fmt.Printf("[DEV-OTP-FALLBACK] email=%s code=%s\n", email, otpCode)
		}
	} else {
		fmt.Printf("[DEV-OTP] email=%s code=%s\n", email, otpCode)
	}

	return nil
}

func (s *Service) VerifyOTP(ctx context.Context, email, code string) (*User, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	code = strings.TrimSpace(code)
	if email == "" || code == "" {
		return nil, ErrInvalidOTP
	}

	guardKey := normalizeGuardKey(email)
	locked, _, err := s.isGuardLocked(ctx, "otp_verify", guardKey)
	if err != nil {
		return nil, fmt.Errorf("check otp guard: %w", err)
	}
	if locked {
		return nil, ErrRateLimited
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	row := tx.QueryRowContext(ctx, `
		SELECT id, user_id, code_hash, expires_at, attempt_count, max_attempt
		FROM auth_otp_codes
		WHERE email = $1
		  AND purpose = 'login'
		  AND used_at IS NULL
		ORDER BY created_at DESC
		LIMIT 1
		FOR UPDATE
	`, email)

	var otpID int64
	var userID int64
	var codeHash string
	var expiresAt time.Time
	var attempts int
	var maxAttempt int
	if err := row.Scan(&otpID, &userID, &codeHash, &expiresAt, &attempts, &maxAttempt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			_ = s.registerFailure(ctx, "otp_verify", guardKey, s.otpMaxFailures, s.otpLockDuration)
			return nil, ErrInvalidOTP
		}
		return nil, fmt.Errorf("load otp: %w", err)
	}

	if time.Now().After(expiresAt) {
		_, _ = tx.ExecContext(ctx, `UPDATE auth_otp_codes SET used_at = now() WHERE id = $1`, otpID)
		_ = s.registerFailure(ctx, "otp_verify", guardKey, s.otpMaxFailures, s.otpLockDuration)
		return nil, ErrOTPExpired
	}
	if attempts >= maxAttempt {
		_ = s.registerFailure(ctx, "otp_verify", guardKey, s.otpMaxFailures, s.otpLockDuration)
		return nil, ErrInvalidOTP
	}

	expected := hashOTP(email, code)
	if !secureEqual(expected, codeHash) {
		_, _ = tx.ExecContext(ctx, `UPDATE auth_otp_codes SET attempt_count = attempt_count + 1 WHERE id = $1`, otpID)
		if err := tx.Commit(); err != nil {
			return nil, fmt.Errorf("commit failed attempt: %w", err)
		}
		_ = s.registerFailure(ctx, "otp_verify", guardKey, s.otpMaxFailures, s.otpLockDuration)
		return nil, ErrInvalidOTP
	}

	_, err = tx.ExecContext(ctx, `UPDATE auth_otp_codes SET used_at = now() WHERE id = $1`, otpID)
	if err != nil {
		return nil, fmt.Errorf("consume otp: %w", err)
	}

	user, err := s.getActiveUserTx(ctx, tx, userID)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit otp: %w", err)
	}

	_ = s.clearGuard(ctx, "otp_verify", guardKey)
	return user, nil
}

func (s *Service) BootstrapAccounts(ctx context.Context, in BootstrapInput) error {
	if strings.TrimSpace(s.bootstrapToken) == "" {
		return ErrBootstrapDenied
	}
	if in.Token != s.bootstrapToken {
		return ErrBootstrapDenied
	}

	admin := BootstrapInput{
		AdminUsername: strings.TrimSpace(in.AdminUsername),
		AdminEmail:    strings.ToLower(strings.TrimSpace(in.AdminEmail)),
		AdminPassword: strings.TrimSpace(in.AdminPassword),
	}
	proktor := BootstrapInput{
		ProktorUsername: strings.TrimSpace(in.ProktorUsername),
		ProktorEmail:    strings.ToLower(strings.TrimSpace(in.ProktorEmail)),
		ProktorPassword: strings.TrimSpace(in.ProktorPassword),
	}

	if admin.AdminUsername == "" || admin.AdminEmail == "" || admin.AdminPassword == "" {
		return errors.New("admin bootstrap fields are required")
	}
	if proktor.ProktorUsername == "" || proktor.ProktorEmail == "" || proktor.ProktorPassword == "" {
		return errors.New("proktor bootstrap fields are required")
	}
	if _, err := mail.ParseAddress(admin.AdminEmail); err != nil {
		return errors.New("invalid admin email")
	}
	if _, err := mail.ParseAddress(proktor.ProktorEmail); err != nil {
		return errors.New("invalid proktor email")
	}
	if len(admin.AdminPassword) < 8 || len(proktor.ProktorPassword) < 8 {
		return errors.New("bootstrap passwords must be at least 8 characters")
	}

	adminHash, err := bcrypt.GenerateFromPassword([]byte(admin.AdminPassword), s.bcryptCost)
	if err != nil {
		return fmt.Errorf("hash admin password: %w", err)
	}
	proktorHash, err := bcrypt.GenerateFromPassword([]byte(proktor.ProktorPassword), s.bcryptCost)
	if err != nil {
		return fmt.Errorf("hash proktor password: %w", err)
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin bootstrap tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if err := s.upsertBootstrapUser(ctx, tx, admin.AdminUsername, admin.AdminEmail, string(adminHash), "Administrator CBT", "admin"); err != nil {
		return err
	}
	if err := s.upsertBootstrapUser(ctx, tx, proktor.ProktorUsername, proktor.ProktorEmail, string(proktorHash), "Proktor CBT", "proktor"); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit bootstrap: %w", err)
	}
	return nil
}

func (s *Service) upsertBootstrapUser(ctx context.Context, tx *sql.Tx, username, email, passwordHash, fullName, role string) error {
	_, err := tx.ExecContext(ctx, `
		INSERT INTO users (
			username, password_hash, full_name, role, is_active,
			email, email_verified_at, account_status, approved_at, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, TRUE,
			$5, now(), 'active', now(), now(), now()
		)
		ON CONFLICT (username)
		DO UPDATE SET
			password_hash = EXCLUDED.password_hash,
			full_name = EXCLUDED.full_name,
			role = EXCLUDED.role,
			is_active = TRUE,
			email = EXCLUDED.email,
			email_verified_at = now(),
			account_status = 'active',
			approved_at = now(),
			updated_at = now()
	`, username, passwordHash, fullName, role, email)
	if err != nil {
		return fmt.Errorf("upsert bootstrap user %s: %w", username, err)
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO auth_identities (user_id, provider, provider_key, created_at)
		SELECT id, 'password', username, now()
		FROM users
		WHERE username = $1
		ON CONFLICT (provider, provider_key) DO NOTHING
	`, username)
	if err != nil {
		return fmt.Errorf("upsert bootstrap identity %s: %w", username, err)
	}

	return nil
}

func (s *Service) CreateRegistration(ctx context.Context, in RegistrationInput) (int64, error) {
	in.RoleRequested = strings.TrimSpace(in.RoleRequested)
	in.Email = strings.ToLower(strings.TrimSpace(in.Email))
	in.FullName = strings.TrimSpace(in.FullName)
	in.Phone = strings.TrimSpace(in.Phone)
	in.InstitutionName = strings.TrimSpace(in.InstitutionName)

	if in.RoleRequested != "siswa" && in.RoleRequested != "guru" {
		return 0, errors.New("role_requested must be siswa or guru")
	}
	if _, err := mail.ParseAddress(in.Email); err != nil {
		return 0, errors.New("invalid email")
	}
	if len(in.Password) < 8 {
		return 0, errors.New("password must be at least 8 characters")
	}
	if in.FullName == "" {
		return 0, errors.New("full_name is required")
	}
	if strings.TrimSpace(in.FormPayload) == "" {
		in.FormPayload = "{}"
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(in.Password), s.bcryptCost)
	if err != nil {
		return 0, fmt.Errorf("hash password: %w", err)
	}

	var id int64
	err = s.db.QueryRowContext(ctx, `
		INSERT INTO registration_requests (
			role_requested,
			email,
			password_hash,
			full_name,
			phone,
			institution_name,
			form_payload,
			status,
			created_at,
			updated_at
		) VALUES (
			$1,$2,$3,$4,$5,$6,$7::jsonb,'pending',now(),now()
		)
		RETURNING id
	`, in.RoleRequested, in.Email, string(passwordHash), in.FullName, nullableString(in.Phone), nullableString(in.InstitutionName), in.FormPayload).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("insert registration: %w", err)
	}
	return id, nil
}

func (s *Service) ListRegistrations(ctx context.Context, status string, limit, offset int) ([]RegistrationRecord, error) {
	status = strings.ToLower(strings.TrimSpace(status))
	switch status {
	case "", "pending", "approved", "rejected":
	default:
		return nil, errors.New("invalid status filter")
	}
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT id, role_requested, email, full_name, phone, institution_name, status, review_note, reviewed_by, reviewed_at, created_at
		FROM registration_requests
		WHERE ($1 = '' OR status = $1)
		ORDER BY created_at DESC, id DESC
		LIMIT $2
		OFFSET $3
	`, status, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list registrations: %w", err)
	}
	defer rows.Close()

	out := make([]RegistrationRecord, 0, limit)
	for rows.Next() {
		var r RegistrationRecord
		var phone sql.NullString
		var institution sql.NullString
		var reviewNote sql.NullString
		var reviewedBy sql.NullInt64
		var reviewedAt sql.NullTime
		if err := rows.Scan(&r.ID, &r.RoleRequested, &r.Email, &r.FullName, &phone, &institution, &r.Status, &reviewNote, &reviewedBy, &reviewedAt, &r.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan registration: %w", err)
		}
		if phone.Valid {
			r.Phone = &phone.String
		}
		if institution.Valid {
			r.Institution = &institution.String
		}
		if reviewNote.Valid {
			r.ReviewNote = &reviewNote.String
		}
		if reviewedBy.Valid {
			r.ReviewedBy = &reviewedBy.Int64
		}
		if reviewedAt.Valid {
			r.ReviewedAt = &reviewedAt.Time
		}
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate registrations: %w", err)
	}
	return out, nil
}

func (s *Service) ListRegistrationPending(ctx context.Context, limit int) ([]RegistrationRecord, error) {
	return s.ListRegistrations(ctx, "pending", limit, 0)
}

func (s *Service) ApproveRegistration(ctx context.Context, registrationID, reviewerID int64) (int64, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	row := tx.QueryRowContext(ctx, `
		SELECT id, role_requested, email, password_hash, full_name, status
		FROM registration_requests
		WHERE id = $1
		FOR UPDATE
	`, registrationID)

	var id int64
	var role, email, passwordHash, fullName, status string
	if err := row.Scan(&id, &role, &email, &passwordHash, &fullName, &status); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, ErrRegistrationNotFound
		}
		return 0, fmt.Errorf("load registration: %w", err)
	}
	if status != "pending" {
		return 0, ErrRegistrationState
	}

	var userID int64
	err = tx.QueryRowContext(ctx, `SELECT id FROM users WHERE email = $1 LIMIT 1`, email).Scan(&userID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return 0, fmt.Errorf("lookup user email: %w", err)
	}

	if userID == 0 {
		username, err := s.nextUsernameTx(ctx, tx, email)
		if err != nil {
			return 0, err
		}

		err = tx.QueryRowContext(ctx, `
			INSERT INTO users (
				username, password_hash, full_name, role, is_active,
				email, email_verified_at, account_status, approved_by, approved_at, created_at, updated_at
			) VALUES (
				$1, $2, $3, $4, TRUE,
				$5, now(), 'active', $6, now(), now(), now()
			)
			RETURNING id
		`, username, passwordHash, fullName, role, email, reviewerID).Scan(&userID)
		if err != nil {
			return 0, fmt.Errorf("insert approved user: %w", err)
		}
	} else {
		_, err = tx.ExecContext(ctx, `
			UPDATE users
			SET full_name = COALESCE(NULLIF($1,''), full_name),
				password_hash = $2,
				role = $3,
				is_active = TRUE,
				email_verified_at = COALESCE(email_verified_at, now()),
				account_status = 'active',
				approved_by = $4,
				approved_at = now(),
				updated_at = now()
			WHERE id = $5
		`, fullName, passwordHash, role, reviewerID, userID)
		if err != nil {
			return 0, fmt.Errorf("update approved user: %w", err)
		}
	}

	_, err = tx.ExecContext(ctx, `
		UPDATE registration_requests
		SET status = 'approved',
			review_note = 'approved',
			reviewed_by = $1,
			reviewed_at = now(),
			created_user_id = $2,
			updated_at = now()
		WHERE id = $3
	`, reviewerID, userID, registrationID)
	if err != nil {
		return 0, fmt.Errorf("update registration approved: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit approve: %w", err)
	}
	return userID, nil
}

func (s *Service) RejectRegistration(ctx context.Context, registrationID, reviewerID int64, note string) error {
	note = strings.TrimSpace(note)
	if note == "" {
		note = "rejected"
	}

	res, err := s.db.ExecContext(ctx, `
		UPDATE registration_requests
		SET status = 'rejected',
			review_note = $1,
			reviewed_by = $2,
			reviewed_at = now(),
			updated_at = now()
		WHERE id = $3
		  AND status = 'pending'
	`, note, reviewerID, registrationID)
	if err != nil {
		return fmt.Errorf("reject registration: %w", err)
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		return ErrRegistrationState
	}
	return nil
}

func (s *Service) CreateSession(ctx context.Context, userID int64, ipAddress, userAgent string) (string, time.Time, error) {
	token, err := generateToken(32)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("generate session token: %w", err)
	}
	tokenHash := hashToken(token)
	expiresAt := time.Now().Add(s.sessionTTL)

	_, err = s.db.ExecContext(ctx, `
		INSERT INTO auth_sessions (
			user_id, session_token_hash, expires_at, ip_address, user_agent, created_at
		) VALUES (
			$1, $2, $3, $4, $5, now()
		)
	`, userID, tokenHash, expiresAt, nullableString(ipAddress), nullableString(userAgent))
	if err != nil {
		return "", time.Time{}, fmt.Errorf("insert session: %w", err)
	}
	return token, expiresAt, nil
}

func (s *Service) GetSessionUser(ctx context.Context, token string) (*User, error) {
	if strings.TrimSpace(token) == "" {
		return nil, ErrUnauthorized
	}
	row := s.db.QueryRowContext(ctx, `
		SELECT u.id, u.username, u.email, u.full_name, u.role, u.account_status, u.approved_at
		FROM auth_sessions s
		JOIN users u ON u.id = s.user_id
		WHERE s.session_token_hash = $1
		  AND s.revoked_at IS NULL
		  AND s.expires_at > now()
		LIMIT 1
	`, hashToken(token))

	var u User
	var email sql.NullString
	var approvedAt sql.NullTime
	if err := row.Scan(&u.ID, &u.Username, &email, &u.FullName, &u.Role, &u.AccountStatus, &approvedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUnauthorized
		}
		return nil, fmt.Errorf("query session user: %w", err)
	}
	if email.Valid {
		u.Email = &email.String
	}
	if approvedAt.Valid {
		u.ApprovedAt = &approvedAt.Time
	}
	if u.AccountStatus != "active" {
		return nil, ErrUnauthorized
	}
	return &u, nil
}

func (s *Service) RevokeSession(ctx context.Context, token string) error {
	if strings.TrimSpace(token) == "" {
		return nil
	}
	_, err := s.db.ExecContext(ctx, `
		UPDATE auth_sessions
		SET revoked_at = now()
		WHERE session_token_hash = $1
		  AND revoked_at IS NULL
	`, hashToken(token))
	if err != nil {
		return fmt.Errorf("revoke session: %w", err)
	}
	return nil
}

func (s *Service) getActiveUserTx(ctx context.Context, tx *sql.Tx, userID int64) (*User, error) {
	row := tx.QueryRowContext(ctx, `
		SELECT id, username, email, full_name, role, account_status, approved_at
		FROM users
		WHERE id = $1
		LIMIT 1
	`, userID)

	var u User
	var email sql.NullString
	var approvedAt sql.NullTime
	if err := row.Scan(&u.ID, &u.Username, &email, &u.FullName, &u.Role, &u.AccountStatus, &approvedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUnauthorized
		}
		return nil, fmt.Errorf("query active user: %w", err)
	}
	if email.Valid {
		u.Email = &email.String
	}
	if approvedAt.Valid {
		u.ApprovedAt = &approvedAt.Time
	}
	if u.AccountStatus != "active" {
		return nil, ErrForbidden
	}
	return &u, nil
}

func isValidRole(role string) bool {
	switch role {
	case "admin", "proktor", "guru", "siswa":
		return true
	default:
		return false
	}
}

func (s *Service) ListUsers(ctx context.Context, role, q string, limit, offset int) ([]AdminUserRecord, error) {
	role = strings.ToLower(strings.TrimSpace(role))
	if role != "" && !isValidRole(role) {
		return nil, errors.New("invalid role filter")
	}
	q = strings.TrimSpace(q)
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT
			u.id,
			u.username,
			u.email,
			u.full_name,
			u.role,
			sch.school_id,
			sch.name AS school_name,
			sch.class_id,
			sch.class_name,
			u.is_active,
			u.account_status,
			u.approved_at,
			u.created_at
		FROM users u
		LEFT JOIN LATERAL (
			SELECT
				s.name,
				e.school_id,
				e.class_id,
				c.name AS class_name
			FROM enrollments e
			JOIN schools s ON s.id = e.school_id
			LEFT JOIN classes c ON c.id = e.class_id
			WHERE e.user_id = u.id
			ORDER BY e.enrolled_at DESC, e.id DESC
			LIMIT 1
		) sch ON TRUE
		WHERE ($1 = '' OR u.role = $1)
		  AND (
			$2 = ''
			OR u.username ILIKE '%' || $2 || '%'
			OR u.full_name ILIKE '%' || $2 || '%'
			OR COALESCE(u.email,'') ILIKE '%' || $2 || '%'
			OR COALESCE(sch.name,'') ILIKE '%' || $2 || '%'
		  )
		ORDER BY u.created_at DESC, u.id DESC
		LIMIT $3
		OFFSET $4
	`, role, q, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	defer rows.Close()

	out := make([]AdminUserRecord, 0, limit)
	for rows.Next() {
		var it AdminUserRecord
		var email sql.NullString
		var schoolID sql.NullInt64
		var schoolName sql.NullString
		var classID sql.NullInt64
		var className sql.NullString
		var approvedAt sql.NullTime
		if err := rows.Scan(&it.ID, &it.Username, &email, &it.FullName, &it.Role, &schoolID, &schoolName, &classID, &className, &it.IsActive, &it.AccountStatus, &approvedAt, &it.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan user: %w", err)
		}
		if email.Valid {
			it.Email = &email.String
		}
		if schoolID.Valid {
			it.SchoolID = &schoolID.Int64
		}
		if schoolName.Valid {
			it.SchoolName = &schoolName.String
		}
		if classID.Valid {
			it.ClassID = &classID.Int64
		}
		if className.Valid {
			it.ClassName = &className.String
		}
		if approvedAt.Valid {
			it.ApprovedAt = &approvedAt.Time
		}
		out = append(out, it)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate users: %w", err)
	}
	return out, nil
}

func (s *Service) AdminDashboardStats(ctx context.Context) (*AdminDashboardStats, error) {
	out := &AdminDashboardStats{}
	if err := s.db.QueryRowContext(ctx, `
		SELECT
			COUNT(*) FILTER (WHERE role = 'admin' AND is_active = TRUE),
			COUNT(*) FILTER (WHERE role = 'proktor' AND is_active = TRUE),
			COUNT(*) FILTER (WHERE role = 'guru' AND is_active = TRUE),
			COUNT(*) FILTER (WHERE role = 'siswa' AND is_active = TRUE)
		FROM users
	`).Scan(&out.AdminCount, &out.ProktorCount, &out.GuruCount, &out.SiswaCount); err != nil {
		return nil, fmt.Errorf("query user stats: %w", err)
	}
	if err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM schools
		WHERE is_active = TRUE
	`).Scan(&out.SchoolCount); err != nil {
		return nil, fmt.Errorf("query school stats: %w", err)
	}
	return out, nil
}

func (s *Service) CreateUserByAdmin(ctx context.Context, actorID int64, in AdminCreateUserInput) (*AdminUserRecord, error) {
	username := strings.ToLower(strings.TrimSpace(in.Username))
	email := strings.ToLower(strings.TrimSpace(in.Email))
	fullName := strings.TrimSpace(in.FullName)
	role := strings.ToLower(strings.TrimSpace(in.Role))
	if username == "" || fullName == "" || !isValidRole(role) || len(strings.TrimSpace(in.Password)) < 8 {
		return nil, errors.New("username, full_name, role, and password(>=8) are required")
	}
	if email != "" {
		if _, err := mail.ParseAddress(email); err != nil {
			return nil, errors.New("invalid email")
		}
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(in.Password), s.bcryptCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var out AdminUserRecord
	var emailNull sql.NullString
	var approvedAt sql.NullTime
	err = tx.QueryRowContext(ctx, `
		INSERT INTO users (
			username, password_hash, full_name, role, is_active,
			email, email_verified_at, account_status, approved_by, approved_at, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, TRUE,
			NULLIF($5,''), CASE WHEN NULLIF($5,'') IS NOT NULL THEN now() ELSE NULL END,
			'active', $6, now(), now(), now()
		)
		RETURNING id, username, email, full_name, role, is_active, account_status, approved_at, created_at
	`, username, string(hash), fullName, role, email, actorID).Scan(
		&out.ID, &out.Username, &emailNull, &out.FullName, &out.Role, &out.IsActive, &out.AccountStatus, &approvedAt, &out.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}
	if emailNull.Valid {
		out.Email = &emailNull.String
	}
	if approvedAt.Valid {
		out.ApprovedAt = &approvedAt.Time
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO auth_identities (user_id, provider, provider_key, created_at)
		VALUES ($1, 'password', $2, now())
	`, out.ID, username); err != nil {
		return nil, fmt.Errorf("insert auth identity: %w", err)
	}

	if err := upsertUserEnrollmentTx(ctx, tx, out.ID, in.SchoolID, in.ClassID); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit create user: %w", err)
	}
	return &out, nil
}

func (s *Service) UpdateUserByAdmin(ctx context.Context, actorID, userID int64, in AdminUpdateUserInput) (*AdminUserRecord, error) {
	_ = actorID
	fullName := strings.TrimSpace(in.FullName)
	email := strings.ToLower(strings.TrimSpace(in.Email))
	role := strings.ToLower(strings.TrimSpace(in.Role))
	if userID <= 0 || fullName == "" || !isValidRole(role) {
		return nil, errors.New("id, full_name, and valid role are required")
	}
	if email != "" {
		if _, err := mail.ParseAddress(email); err != nil {
			return nil, errors.New("invalid email")
		}
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if strings.TrimSpace(in.Password) != "" {
		if len(strings.TrimSpace(in.Password)) < 8 {
			return nil, errors.New("password must be at least 8 characters")
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(in.Password), s.bcryptCost)
		if err != nil {
			return nil, fmt.Errorf("hash password: %w", err)
		}
		if _, err := tx.ExecContext(ctx, `
			UPDATE users
			SET password_hash = $2,
				updated_at = now()
			WHERE id = $1
		`, userID, string(hash)); err != nil {
			return nil, fmt.Errorf("update password: %w", err)
		}
	}

	var out AdminUserRecord
	var emailNull sql.NullString
	var approvedAt sql.NullTime
	err = tx.QueryRowContext(ctx, `
		UPDATE users
		SET full_name = $2,
			role = $3,
			email = NULLIF($4,''),
			email_verified_at = CASE
				WHEN NULLIF($4,'') IS NOT NULL THEN COALESCE(email_verified_at, now())
				ELSE email_verified_at
			END,
			updated_at = now()
		WHERE id = $1
		RETURNING id, username, email, full_name, role, is_active, account_status, approved_at, created_at
	`, userID, fullName, role, email).Scan(
		&out.ID, &out.Username, &emailNull, &out.FullName, &out.Role, &out.IsActive, &out.AccountStatus, &approvedAt, &out.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("update user: %w", err)
	}
	if emailNull.Valid {
		out.Email = &emailNull.String
	}
	if approvedAt.Valid {
		out.ApprovedAt = &approvedAt.Time
	}

	if err := upsertUserEnrollmentTx(ctx, tx, userID, in.SchoolID, in.ClassID); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit update user: %w", err)
	}
	return &out, nil
}

func (s *Service) DeactivateUserByAdmin(ctx context.Context, actorID, userID int64) error {
	_ = actorID
	if userID <= 0 {
		return errors.New("invalid user id")
	}
	res, err := s.db.ExecContext(ctx, `
		UPDATE users
		SET is_active = FALSE,
			account_status = 'suspended',
			updated_at = now()
		WHERE id = $1
	`, userID)
	if err != nil {
		return fmt.Errorf("deactivate user: %w", err)
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		return ErrUserNotFound
	}
	return nil
}

func upsertUserEnrollmentTx(ctx context.Context, tx *sql.Tx, userID int64, schoolID, classID *int64) error {
	if schoolID == nil && classID == nil {
		return nil
	}
	if (schoolID == nil) != (classID == nil) {
		return errors.New("school_id dan class_id harus diisi berpasangan")
	}

	if *schoolID <= 0 || *classID <= 0 {
		if _, err := tx.ExecContext(ctx, `
			DELETE FROM enrollments
			WHERE user_id = $1
		`, userID); err != nil {
			return fmt.Errorf("clear enrollment: %w", err)
		}
		return nil
	}

	var valid bool
	if err := tx.QueryRowContext(ctx, `
		SELECT EXISTS(
			SELECT 1
			FROM classes c
			JOIN schools s ON s.id = c.school_id
			WHERE c.id = $1
			  AND c.school_id = $2
			  AND c.is_active = TRUE
			  AND s.is_active = TRUE
		)
	`, *classID, *schoolID).Scan(&valid); err != nil {
		return fmt.Errorf("validate enrollment refs: %w", err)
	}
	if !valid {
		return errors.New("kelas tidak valid untuk sekolah yang dipilih")
	}

	if _, err := tx.ExecContext(ctx, `
		DELETE FROM enrollments
		WHERE user_id = $1
	`, userID); err != nil {
		return fmt.Errorf("replace enrollment delete old: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO enrollments (user_id, school_id, class_id, status, enrolled_at, created_at)
		VALUES ($1, $2, $3, 'active', now(), now())
	`, userID, *schoolID, *classID); err != nil {
		return fmt.Errorf("replace enrollment insert: %w", err)
	}
	return nil
}

func (s *Service) isGuardLocked(ctx context.Context, purpose, subjectKey string) (bool, time.Time, error) {
	var lockedUntil sql.NullTime
	err := s.db.QueryRowContext(ctx, `
		SELECT locked_until
		FROM auth_guard_states
		WHERE purpose = $1 AND subject_key = $2
	`, purpose, subjectKey).Scan(&lockedUntil)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, time.Time{}, nil
		}
		return false, time.Time{}, err
	}
	if !lockedUntil.Valid {
		return false, time.Time{}, nil
	}
	if time.Now().Before(lockedUntil.Time) {
		return true, lockedUntil.Time, nil
	}
	return false, lockedUntil.Time, nil
}

func (s *Service) registerFailure(ctx context.Context, purpose, subjectKey string, maxFailures int, lockDuration time.Duration) error {
	var failedCount int
	err := s.db.QueryRowContext(ctx, `
		INSERT INTO auth_guard_states (purpose, subject_key, failed_count, updated_at, created_at)
		VALUES ($1, $2, 1, now(), now())
		ON CONFLICT (purpose, subject_key)
		DO UPDATE SET
			failed_count = auth_guard_states.failed_count + 1,
			updated_at = now()
		RETURNING failed_count
	`, purpose, subjectKey).Scan(&failedCount)
	if err != nil {
		return err
	}

	if failedCount >= maxFailures {
		_, err = s.db.ExecContext(ctx, `
			UPDATE auth_guard_states
			SET locked_until = now() + ($3 || ' seconds')::interval,
				failed_count = 0,
				updated_at = now()
			WHERE purpose = $1 AND subject_key = $2
		`, purpose, subjectKey, int(lockDuration.Seconds()))
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) clearGuard(ctx context.Context, purpose, subjectKey string) error {
	_, err := s.db.ExecContext(ctx, `
		DELETE FROM auth_guard_states
		WHERE purpose = $1 AND subject_key = $2
	`, purpose, subjectKey)
	return err
}

func (s *Service) nextUsernameTx(ctx context.Context, tx *sql.Tx, email string) (string, error) {
	local := strings.Split(email, "@")[0]
	base := normalizeUsername(local)
	if base == "" {
		base = "user"
	}

	candidate := base
	for i := 0; i < 20; i++ {
		var exists bool
		if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM users WHERE username = $1)`, candidate).Scan(&exists); err != nil {
			return "", fmt.Errorf("check username: %w", err)
		}
		if !exists {
			return candidate, nil
		}
		candidate = fmt.Sprintf("%s%d", base, i+1)
	}

	suffix, err := generateToken(3)
	if err != nil {
		return "", err
	}
	return base + suffix, nil
}

func normalizeUsername(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '.' || r == '_' || r == '-':
			b.WriteRune(r)
		}
	}
	return strings.Trim(b.String(), "._-")
}

func normalizeGuardKey(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

func nullableString(s string) interface{} {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	return s
}

func generateToken(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func generateOTPCode(digits int) (string, error) {
	if digits <= 0 {
		digits = 6
	}
	max := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(digits)), nil)
	n, err := rand.Int(rand.Reader, max)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%0*d", digits, n.Int64()), nil
}

func hashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}

func hashOTP(email, code string) string {
	seed := strings.ToLower(strings.TrimSpace(email)) + ":" + strings.TrimSpace(code)
	h := sha256.Sum256([]byte(seed))
	return hex.EncodeToString(h[:])
}

func secureEqual(a, b string) bool {
	ha := sha256.Sum256([]byte(a))
	hb := sha256.Sum256([]byte(b))
	return ha == hb
}
