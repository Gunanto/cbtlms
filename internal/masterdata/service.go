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

type UpdateSchoolInput struct {
	Name    string
	Code    string
	Address string
}

type UpdateClassInput struct {
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

type ClassListItem struct {
	ID         int64  `json:"id"`
	SchoolID   int64  `json:"school_id"`
	Name       string `json:"name"`
	GradeLevel string `json:"grade_level"`
	IsActive   bool   `json:"is_active"`
}

type EducationLevel struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	IsActive bool   `json:"is_active"`
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

func (s *Service) UpdateSchool(ctx context.Context, actorID, id int64, in UpdateSchoolInput) (*School, error) {
	name := strings.TrimSpace(in.Name)
	code := strings.TrimSpace(in.Code)
	address := strings.TrimSpace(in.Address)
	if id <= 0 || name == "" {
		return nil, ErrInvalidInput
	}

	var out School
	err := s.db.QueryRowContext(ctx, `
		UPDATE schools
		SET name = $2,
			code = NULLIF($3,''),
			address = NULLIF($4,''),
			updated_at = now()
		WHERE id = $1
		RETURNING id, name, COALESCE(code,''), COALESCE(address,'')
	`, id, name, code, address).Scan(&out.ID, &out.Name, &out.Code, &out.Address)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, sql.ErrNoRows
		}
		return nil, fmt.Errorf("update school: %w", err)
	}

	_ = s.writeAudit(ctx, actorID, "school_updated", "school", fmt.Sprintf("%d", out.ID), map[string]any{
		"name": out.Name,
		"code": out.Code,
	})
	return &out, nil
}

func (s *Service) DeleteSchool(ctx context.Context, actorID, id int64) error {
	if id <= 0 {
		return ErrInvalidInput
	}

	res, err := s.db.ExecContext(ctx, `
		DELETE FROM schools
		WHERE id = $1
	`, id)
	if err != nil {
		return fmt.Errorf("delete school: %w", err)
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		return sql.ErrNoRows
	}

	_ = s.writeAudit(ctx, actorID, "school_deleted", "school", fmt.Sprintf("%d", id), map[string]any{})
	return nil
}

func (s *Service) UpdateClass(ctx context.Context, actorID, id int64, in UpdateClassInput) (*Class, error) {
	name := strings.TrimSpace(in.Name)
	grade := strings.TrimSpace(in.GradeLevel)
	if id <= 0 || in.SchoolID <= 0 || name == "" || grade == "" {
		return nil, ErrInvalidInput
	}

	var out Class
	err := s.db.QueryRowContext(ctx, `
		UPDATE classes
		SET school_id = $2,
			name = $3,
			grade_level = $4,
			updated_at = now()
		WHERE id = $1
		RETURNING id, school_id, name, grade_level
	`, id, in.SchoolID, name, grade).Scan(&out.ID, &out.SchoolID, &out.Name, &out.GradeLevel)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, sql.ErrNoRows
		}
		return nil, fmt.Errorf("update class: %w", err)
	}

	_ = s.writeAudit(ctx, actorID, "class_updated", "class", fmt.Sprintf("%d", out.ID), map[string]any{
		"school_id":   out.SchoolID,
		"name":        out.Name,
		"grade_level": out.GradeLevel,
	})
	return &out, nil
}

func (s *Service) DeleteClass(ctx context.Context, actorID, id int64) error {
	if id <= 0 {
		return ErrInvalidInput
	}

	res, err := s.db.ExecContext(ctx, `
		DELETE FROM classes
		WHERE id = $1
	`, id)
	if err != nil {
		return fmt.Errorf("delete class: %w", err)
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		return sql.ErrNoRows
	}

	_ = s.writeAudit(ctx, actorID, "class_deleted", "class", fmt.Sprintf("%d", id), map[string]any{})
	return nil
}

func (s *Service) ListSchools(ctx context.Context, activeOnly bool) ([]School, error) {
	query := `
		SELECT id, name, COALESCE(code,''), COALESCE(address,'')
		FROM schools
	`
	if activeOnly {
		query += " WHERE is_active = TRUE"
	}
	query += " ORDER BY name ASC, id ASC"

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list schools: %w", err)
	}
	defer rows.Close()

	out := make([]School, 0)
	for rows.Next() {
		var it School
		if err := rows.Scan(&it.ID, &it.Name, &it.Code, &it.Address); err != nil {
			return nil, fmt.Errorf("scan school: %w", err)
		}
		out = append(out, it)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate schools: %w", err)
	}
	return out, nil
}

func (s *Service) ListClasses(ctx context.Context, schoolID int64, activeOnly bool) ([]ClassListItem, error) {
	query := `
		SELECT id, school_id, name, grade_level, is_active
		FROM classes
		WHERE ($1 <= 0 OR school_id = $1)
	`
	if activeOnly {
		query += " AND is_active = TRUE"
	}
	query += " ORDER BY grade_level ASC, name ASC, id ASC"

	rows, err := s.db.QueryContext(ctx, query, schoolID)
	if err != nil {
		return nil, fmt.Errorf("list classes: %w", err)
	}
	defer rows.Close()

	out := make([]ClassListItem, 0)
	for rows.Next() {
		var it ClassListItem
		if err := rows.Scan(&it.ID, &it.SchoolID, &it.Name, &it.GradeLevel, &it.IsActive); err != nil {
			return nil, fmt.Errorf("scan class: %w", err)
		}
		out = append(out, it)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate classes: %w", err)
	}
	return out, nil
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

func (s *Service) ListEducationLevels(ctx context.Context, activeOnly bool) ([]EducationLevel, error) {
	query := `
		SELECT id, name, is_active
		FROM education_levels
	`
	if activeOnly {
		query += " WHERE is_active = TRUE"
	}
	query += " ORDER BY name ASC"

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list education levels: %w", err)
	}
	defer rows.Close()

	out := make([]EducationLevel, 0)
	for rows.Next() {
		var it EducationLevel
		if err := rows.Scan(&it.ID, &it.Name, &it.IsActive); err != nil {
			return nil, fmt.Errorf("scan education level: %w", err)
		}
		out = append(out, it)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate education levels: %w", err)
	}
	return out, nil
}

func (s *Service) CreateEducationLevel(ctx context.Context, actorID int64, name string) (*EducationLevel, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, ErrInvalidInput
	}

	var out EducationLevel
	err := s.db.QueryRowContext(ctx, `
		INSERT INTO education_levels (name, is_active, created_at, updated_at)
		VALUES ($1, TRUE, now(), now())
		RETURNING id, name, is_active
	`, name).Scan(&out.ID, &out.Name, &out.IsActive)
	if err != nil {
		return nil, fmt.Errorf("create education level: %w", err)
	}

	_ = s.writeAudit(ctx, actorID, "education_level_created", "education_level", fmt.Sprintf("%d", out.ID), map[string]any{
		"name": out.Name,
	})
	return &out, nil
}

func (s *Service) UpdateEducationLevel(ctx context.Context, actorID, id int64, name string) (*EducationLevel, error) {
	name = strings.TrimSpace(name)
	if id <= 0 || name == "" {
		return nil, ErrInvalidInput
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var oldName string
	if err := tx.QueryRowContext(ctx, `
		SELECT name FROM education_levels WHERE id = $1
	`, id).Scan(&oldName); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, sql.ErrNoRows
		}
		return nil, fmt.Errorf("load education level: %w", err)
	}

	var out EducationLevel
	if err := tx.QueryRowContext(ctx, `
		UPDATE education_levels
		SET name = $2,
			updated_at = now()
		WHERE id = $1
		RETURNING id, name, is_active
	`, id, name).Scan(&out.ID, &out.Name, &out.IsActive); err != nil {
		return nil, fmt.Errorf("update education level: %w", err)
	}

	if strings.TrimSpace(oldName) != strings.TrimSpace(name) {
		if _, err := tx.ExecContext(ctx, `
			UPDATE subjects
			SET education_level = $2
			WHERE education_level = $1
		`, oldName, name); err != nil {
			return nil, fmt.Errorf("sync subjects education_level: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit tx: %w", err)
	}

	_ = s.writeAudit(ctx, actorID, "education_level_updated", "education_level", fmt.Sprintf("%d", out.ID), map[string]any{
		"old_name": oldName,
		"name":     out.Name,
	})
	return &out, nil
}

func (s *Service) DeleteEducationLevel(ctx context.Context, actorID, id int64) error {
	if id <= 0 {
		return ErrInvalidInput
	}

	var name string
	if err := s.db.QueryRowContext(ctx, `
		SELECT name FROM education_levels WHERE id = $1
	`, id).Scan(&name); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return sql.ErrNoRows
		}
		return fmt.Errorf("load education level: %w", err)
	}

	var subjectCount int
	if err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(1) FROM subjects WHERE education_level = $1
	`, name).Scan(&subjectCount); err != nil {
		return fmt.Errorf("count subject references: %w", err)
	}
	if subjectCount > 0 {
		return errors.New("education level is still used by subjects")
	}

	res, err := s.db.ExecContext(ctx, `
		DELETE FROM education_levels WHERE id = $1
	`, id)
	if err != nil {
		return fmt.Errorf("delete education level: %w", err)
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		return sql.ErrNoRows
	}

	_ = s.writeAudit(ctx, actorID, "education_level_deleted", "education_level", fmt.Sprintf("%d", id), map[string]any{
		"name": name,
	})
	return nil
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
