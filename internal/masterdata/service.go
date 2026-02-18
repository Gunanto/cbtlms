package masterdata

import (
	"context"
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/mail"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidInput = errors.New("invalid input")
)

type Service struct {
	db         *sql.DB
	bcryptCost int
}

type CreateSchoolInput struct {
	Name    string
	Code    string
	Address string
}

type School struct {
	ID      int64  `json:"id"`
	Name    string `json:"name"`
	Code    string `json:"code,omitempty"`
	Address string `json:"address,omitempty"`
}

type CreateClassInput struct {
	SchoolID   int64
	Name       string
	GradeLevel string
}

type Class struct {
	ID         int64  `json:"id"`
	SchoolID   int64  `json:"school_id"`
	Name       string `json:"name"`
	GradeLevel string `json:"grade_level"`
}

type ImportStudentsReport struct {
	TotalRows   int              `json:"total_rows"`
	SuccessRows int              `json:"success_rows"`
	FailedRows  int              `json:"failed_rows"`
	Errors      []ImportRowError `json:"errors"`
}

type ImportRowError struct {
	Row   int    `json:"row"`
	Error string `json:"error"`
}

func NewService(db *sql.DB) *Service {
	return &Service{db: db, bcryptCost: bcrypt.DefaultCost}
}

func (s *Service) CreateSchool(ctx context.Context, actorID int64, in CreateSchoolInput) (*School, error) {
	name := strings.TrimSpace(in.Name)
	code := strings.TrimSpace(in.Code)
	address := strings.TrimSpace(in.Address)
	if name == "" {
		return nil, ErrInvalidInput
	}

	var school School
	err := s.db.QueryRowContext(ctx, `
		INSERT INTO schools (name, code, address, created_at, updated_at)
		VALUES ($1, NULLIF($2,''), NULLIF($3,''), now(), now())
		RETURNING id, name, COALESCE(code,''), COALESCE(address,'')
	`, name, code, address).Scan(&school.ID, &school.Name, &school.Code, &school.Address)
	if err != nil {
		return nil, fmt.Errorf("create school: %w", err)
	}

	_ = s.writeAudit(ctx, actorID, "school_created", "school", fmt.Sprintf("%d", school.ID), map[string]any{
		"name": school.Name,
		"code": school.Code,
	})

	return &school, nil
}

func (s *Service) CreateClass(ctx context.Context, actorID int64, in CreateClassInput) (*Class, error) {
	name := strings.TrimSpace(in.Name)
	grade := strings.TrimSpace(in.GradeLevel)
	if in.SchoolID <= 0 || name == "" || grade == "" {
		return nil, ErrInvalidInput
	}

	var class Class
	err := s.db.QueryRowContext(ctx, `
		INSERT INTO classes (school_id, name, grade_level, created_at, updated_at)
		VALUES ($1, $2, $3, now(), now())
		RETURNING id, school_id, name, grade_level
	`, in.SchoolID, name, grade).Scan(&class.ID, &class.SchoolID, &class.Name, &class.GradeLevel)
	if err != nil {
		return nil, fmt.Errorf("create class: %w", err)
	}

	_ = s.writeAudit(ctx, actorID, "class_created", "class", fmt.Sprintf("%d", class.ID), map[string]any{
		"school_id":   class.SchoolID,
		"name":        class.Name,
		"grade_level": class.GradeLevel,
	})

	return &class, nil
}

func (s *Service) ImportStudentsCSV(ctx context.Context, actorID int64, r io.Reader) (*ImportStudentsReport, error) {
	reader := csv.NewReader(r)
	reader.TrimLeadingSpace = true

	header, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("read csv header: %w", err)
	}
	index := make(map[string]int, len(header))
	for i, h := range header {
		n := normalizeHeader(h)
		if n != "" {
			index[n] = i
		}
	}

	required := []string{"full_name", "username", "password", "school_name", "class_name", "grade_level"}
	for _, col := range required {
		if _, ok := index[col]; !ok {
			return nil, fmt.Errorf("missing required column: %s", col)
		}
	}

	report := &ImportStudentsReport{Errors: make([]ImportRowError, 0)}
	rowNo := 1
	for {
		rowNo++
		rec, err := reader.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			report.TotalRows++
			report.FailedRows++
			report.Errors = append(report.Errors, ImportRowError{Row: rowNo, Error: fmt.Sprintf("csv parse error: %v", err)})
			continue
		}

		report.TotalRows++
		if isRowEmpty(rec) {
			continue
		}

		row := map[string]string{
			"full_name":   cell(rec, index, "full_name"),
			"username":    strings.ToLower(cell(rec, index, "username")),
			"password":    cell(rec, index, "password"),
			"email":       strings.ToLower(cell(rec, index, "email")),
			"nisn":        cell(rec, index, "nisn"),
			"school_name": cell(rec, index, "school_name"),
			"class_name":  cell(rec, index, "class_name"),
			"grade_level": cell(rec, index, "grade_level"),
		}

		if err := validateImportRow(row); err != nil {
			report.FailedRows++
			report.Errors = append(report.Errors, ImportRowError{Row: rowNo, Error: err.Error()})
			continue
		}

		if err := s.importStudentRow(ctx, row); err != nil {
			report.FailedRows++
			report.Errors = append(report.Errors, ImportRowError{Row: rowNo, Error: err.Error()})
			continue
		}
		report.SuccessRows++
	}

	_ = s.writeAudit(ctx, actorID, "students_import_csv", "student_import", "csv", map[string]any{
		"total_rows":   report.TotalRows,
		"success_rows": report.SuccessRows,
		"failed_rows":  report.FailedRows,
	})

	return report, nil
}

func (s *Service) importStudentRow(ctx context.Context, row map[string]string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	schoolID, err := getOrCreateSchoolTx(ctx, tx, row["school_name"])
	if err != nil {
		return err
	}
	classID, err := getOrCreateClassTx(ctx, tx, schoolID, row["class_name"], row["grade_level"])
	if err != nil {
		return err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(row["password"]), s.bcryptCost)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	var email any
	var verifiedAt any
	if row["email"] != "" {
		email = row["email"]
		now := time.Now()
		verifiedAt = now
	}

	var userID int64
	err = tx.QueryRowContext(ctx, `
		INSERT INTO users (
			username, password_hash, full_name, role, is_active,
			email, email_verified_at, account_status, approved_at, created_at, updated_at
		) VALUES (
			$1, $2, $3, 'siswa', TRUE,
			$4, $5, 'active', now(), now(), now()
		)
		RETURNING id
	`, row["username"], string(hash), row["full_name"], email, verifiedAt).Scan(&userID)
	if err != nil {
		return fmt.Errorf("insert user: %w", err)
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO auth_identities (user_id, provider, provider_key, created_at)
		VALUES ($1, 'password', $2, now())
		ON CONFLICT (provider, provider_key) DO NOTHING
	`, userID, row["username"])
	if err != nil {
		return fmt.Errorf("insert identity: %w", err)
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO student_profiles (user_id, nisn, school_id, grade_level, class_name)
		VALUES ($1, NULLIF($2,''), $3, $4, $5)
		ON CONFLICT (user_id)
		DO UPDATE SET
			nisn = EXCLUDED.nisn,
			school_id = EXCLUDED.school_id,
			grade_level = EXCLUDED.grade_level,
			class_name = EXCLUDED.class_name
	`, userID, row["nisn"], schoolID, row["grade_level"], row["class_name"])
	if err != nil {
		return fmt.Errorf("upsert student profile: %w", err)
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO enrollments (user_id, school_id, class_id, status, enrolled_at, created_at)
		VALUES ($1, $2, $3, 'active', now(), now())
		ON CONFLICT (user_id, class_id) DO NOTHING
	`, userID, schoolID, classID)
	if err != nil {
		return fmt.Errorf("insert enrollment: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}
	return nil
}

func getOrCreateSchoolTx(ctx context.Context, tx *sql.Tx, schoolName string) (int64, error) {
	var schoolID int64
	err := tx.QueryRowContext(ctx, `
		SELECT id
		FROM schools
		WHERE LOWER(name) = LOWER($1)
		LIMIT 1
	`, schoolName).Scan(&schoolID)
	if err == nil {
		return schoolID, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return 0, fmt.Errorf("lookup school: %w", err)
	}

	err = tx.QueryRowContext(ctx, `
		INSERT INTO schools (name, created_at, updated_at)
		VALUES ($1, now(), now())
		RETURNING id
	`, schoolName).Scan(&schoolID)
	if err != nil {
		return 0, fmt.Errorf("insert school: %w", err)
	}
	return schoolID, nil
}

func getOrCreateClassTx(ctx context.Context, tx *sql.Tx, schoolID int64, className, gradeLevel string) (int64, error) {
	var classID int64
	err := tx.QueryRowContext(ctx, `
		SELECT id
		FROM classes
		WHERE school_id = $1 AND name = $2 AND grade_level = $3
		LIMIT 1
	`, schoolID, className, gradeLevel).Scan(&classID)
	if err == nil {
		return classID, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return 0, fmt.Errorf("lookup class: %w", err)
	}

	err = tx.QueryRowContext(ctx, `
		INSERT INTO classes (school_id, name, grade_level, created_at, updated_at)
		VALUES ($1, $2, $3, now(), now())
		RETURNING id
	`, schoolID, className, gradeLevel).Scan(&classID)
	if err != nil {
		return 0, fmt.Errorf("insert class: %w", err)
	}
	return classID, nil
}

func (s *Service) writeAudit(ctx context.Context, userID int64, action, entityType, entityID string, payload map[string]any) error {
	b, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO audit_logs (user_id, action, entity_type, entity_id, payload, created_at)
		VALUES ($1, $2, $3, $4, $5::jsonb, now())
	`, userID, action, entityType, entityID, string(b))
	return err
}

func normalizeHeader(h string) string {
	h = strings.ToLower(strings.TrimSpace(h))
	h = strings.ReplaceAll(h, "-", "_")
	h = strings.ReplaceAll(h, " ", "_")
	return h
}

func cell(rec []string, idx map[string]int, key string) string {
	i, ok := idx[key]
	if !ok || i < 0 || i >= len(rec) {
		return ""
	}
	return strings.TrimSpace(rec[i])
}

func validateImportRow(row map[string]string) error {
	if row["full_name"] == "" {
		return errors.New("full_name is required")
	}
	if row["username"] == "" {
		return errors.New("username is required")
	}
	if len(row["password"]) < 8 {
		return errors.New("password minimum length is 8")
	}
	if row["school_name"] == "" || row["class_name"] == "" || row["grade_level"] == "" {
		return errors.New("school_name, class_name, and grade_level are required")
	}
	if row["email"] != "" {
		if _, err := mail.ParseAddress(row["email"]); err != nil {
			return errors.New("invalid email format")
		}
	}
	return nil
}

func isRowEmpty(rec []string) bool {
	for _, c := range rec {
		if strings.TrimSpace(c) != "" {
			return false
		}
	}
	return true
}
