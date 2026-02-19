package auth

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"net/mail"
	"strconv"
	"strings"

	"github.com/xuri/excelize/v2"
)

type UserImportRowError struct {
	Row      int    `json:"row"`
	Username string `json:"username,omitempty"`
	Error    string `json:"error"`
}

type UserImportReport struct {
	TotalRows   int                  `json:"total_rows"`
	SuccessRows int                  `json:"success_rows"`
	FailedRows  int                  `json:"failed_rows"`
	Errors      []UserImportRowError `json:"errors"`
}

func (s *Service) ExportUsersExcel(ctx context.Context, role, q string) ([]byte, error) {
	items, err := s.ListUsers(ctx, role, q, 10000, 0)
	if err != nil {
		return nil, err
	}

	f := excelize.NewFile()
	sheet := f.GetSheetName(0)
	headers := []string{
		"username",
		"email",
		"full_name",
		"role",
		"school_name",
		"class_name",
		"grade_level",
		"is_active",
		"account_status",
		"created_at",
	}
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		_ = f.SetCellValue(sheet, cell, h)
	}
	for i, it := range items {
		row := i + 2
		email := ""
		if it.Email != nil {
			email = *it.Email
		}
		schoolName := ""
		if it.SchoolName != nil {
			schoolName = strings.TrimSpace(*it.SchoolName)
		}
		className := ""
		if it.ClassName != nil {
			className = strings.TrimSpace(*it.ClassName)
		}
		gradeLevel := ""
		if it.ClassGrade != nil {
			gradeLevel = strings.TrimSpace(*it.ClassGrade)
		}
		values := []any{
			it.Username,
			email,
			it.FullName,
			it.Role,
			schoolName,
			className,
			gradeLevel,
			it.IsActive,
			it.AccountStatus,
			it.CreatedAt.Format("2006-01-02 15:04:05"),
		}
		for col, v := range values {
			cell, _ := excelize.CoordinatesToCellName(col+1, row)
			_ = f.SetCellValue(sheet, cell, v)
		}
	}
	_ = f.SetColWidth(sheet, "A", "J", 22)

	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		return nil, fmt.Errorf("write excel: %w", err)
	}
	return buf.Bytes(), nil
}

func (s *Service) ExportUserImportTemplateExcel() ([]byte, error) {
	f := excelize.NewFile()
	sheet := f.GetSheetName(0)
	headers := []string{
		"username",
		"email",
		"full_name",
		"role",
		"password",
		"school_name",
		"class_name",
		"grade_level",
		"is_active",
	}
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		_ = f.SetCellValue(sheet, cell, h)
	}

	exampleRows := [][]any{
		{"siswa_demo_001", "siswa001@example.com", "Siswa Demo 001", "siswa", "password123", "SMPN 1 Punggur", "TKA Matematika", "Kelas 9", "true"},
		{"guru_demo_001", "guru001@example.com", "Guru Demo 001", "guru", "password123", "", "", "", "true"},
	}
	for i, row := range exampleRows {
		rowNo := i + 2
		for j, v := range row {
			cell, _ := excelize.CoordinatesToCellName(j+1, rowNo)
			_ = f.SetCellValue(sheet, cell, v)
		}
	}

	noteRow := len(exampleRows) + 4
	_ = f.SetCellValue(sheet, "A"+strconv.Itoa(noteRow), "Catatan:")
	_ = f.SetCellValue(sheet, "A"+strconv.Itoa(noteRow+1), "- role siswa wajib isi school_name, class_name, grade_level.")
	_ = f.SetCellValue(sheet, "A"+strconv.Itoa(noteRow+2), "- school_name dan class_name+grade_level harus sudah ada di Data Master.")
	_ = f.SetCellValue(sheet, "A"+strconv.Itoa(noteRow+3), "- role admin/proktor/guru boleh kosongkan kolom kelas.")

	_ = f.SetColWidth(sheet, "A", "I", 24)

	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		return nil, fmt.Errorf("write import template excel: %w", err)
	}
	return buf.Bytes(), nil
}

func (s *Service) ImportUsersExcel(ctx context.Context, actorID int64, r io.Reader) (*UserImportReport, error) {
	f, err := excelize.OpenReader(r)
	if err != nil {
		return nil, fmt.Errorf("open excel: %w", err)
	}
	defer func() { _ = f.Close() }()

	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		return nil, errors.New("excel sheet is empty")
	}
	rows, err := f.GetRows(sheets[0])
	if err != nil {
		return nil, fmt.Errorf("read rows: %w", err)
	}
	if len(rows) < 2 {
		return nil, errors.New("no data rows found")
	}

	header := map[string]int{}
	for i, h := range rows[0] {
		header[strings.ToLower(strings.TrimSpace(h))] = i
	}
	required := []string{"username", "full_name", "role"}
	for _, col := range required {
		if _, ok := header[col]; !ok {
			return nil, fmt.Errorf("missing required column: %s", col)
		}
	}

	report := &UserImportReport{Errors: make([]UserImportRowError, 0)}
	for i := 1; i < len(rows); i++ {
		rowNo := i + 1
		row := rows[i]
		report.TotalRows++

		get := func(key string) string {
			idx, ok := header[key]
			if !ok || idx >= len(row) {
				return ""
			}
			return strings.TrimSpace(row[idx])
		}

		username := strings.ToLower(get("username"))
		fullName := get("full_name")
		role := strings.ToLower(get("role"))
		email := strings.ToLower(get("email"))
		password := get("password")
		activeRaw := strings.ToLower(get("is_active"))
		schoolName := get("school_name")
		className := get("class_name")
		gradeLevel := get("grade_level")

		if username == "" || fullName == "" || !isValidRole(role) {
			report.FailedRows++
			report.Errors = append(report.Errors, UserImportRowError{
				Row:      rowNo,
				Username: username,
				Error:    "username/full_name/role tidak valid",
			})
			continue
		}
		if email != "" {
			if _, err := mail.ParseAddress(email); err != nil {
				report.FailedRows++
				report.Errors = append(report.Errors, UserImportRowError{
					Row:      rowNo,
					Username: username,
					Error:    "email tidak valid",
				})
				continue
			}
		}
		var schoolID *int64
		var classID *int64
		isSiswa := role == "siswa"
		containsPlacement := strings.TrimSpace(schoolName) != "" || strings.TrimSpace(className) != "" || strings.TrimSpace(gradeLevel) != ""
		if isSiswa || containsPlacement {
			if strings.TrimSpace(schoolName) == "" || strings.TrimSpace(className) == "" || strings.TrimSpace(gradeLevel) == "" {
				report.FailedRows++
				report.Errors = append(report.Errors, UserImportRowError{
					Row:      rowNo,
					Username: username,
					Error:    "untuk role siswa, school_name, class_name, grade_level wajib diisi",
				})
				continue
			}
			sid, cid, resolveErr := s.resolveEnrollmentFromNames(ctx, schoolName, className, gradeLevel)
			if resolveErr != nil {
				report.FailedRows++
				report.Errors = append(report.Errors, UserImportRowError{
					Row:      rowNo,
					Username: username,
					Error:    resolveErr.Error(),
				})
				continue
			}
			schoolID = &sid
			classID = &cid
		}

		var userID int64
		err := s.db.QueryRowContext(ctx, `SELECT id FROM users WHERE username = $1`, username).Scan(&userID)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			report.FailedRows++
			report.Errors = append(report.Errors, UserImportRowError{
				Row:      rowNo,
				Username: username,
				Error:    "gagal cek user existing",
			})
			continue
		}

		if userID == 0 {
			if len(strings.TrimSpace(password)) < 8 {
				report.FailedRows++
				report.Errors = append(report.Errors, UserImportRowError{
					Row:      rowNo,
					Username: username,
					Error:    "password minimal 8 karakter untuk user baru",
				})
				continue
			}
			created, err := s.CreateUserByAdmin(ctx, actorID, AdminCreateUserInput{
				Username: username,
				Email:    email,
				Password: password,
				FullName: fullName,
				Role:     role,
				SchoolID: schoolID,
				ClassID:  classID,
			})
			if err != nil {
				report.FailedRows++
				report.Errors = append(report.Errors, UserImportRowError{
					Row:      rowNo,
					Username: username,
					Error:    err.Error(),
				})
				continue
			}
			userID = created.ID
		} else {
			if _, err := s.UpdateUserByAdmin(ctx, actorID, userID, AdminUpdateUserInput{
				FullName: fullName,
				Email:    email,
				Role:     role,
				Password: password,
				SchoolID: schoolID,
				ClassID:  classID,
			}); err != nil {
				report.FailedRows++
				report.Errors = append(report.Errors, UserImportRowError{
					Row:      rowNo,
					Username: username,
					Error:    err.Error(),
				})
				continue
			}
		}

		if activeRaw != "" {
			isActive := parseBoolLoose(activeRaw)
			if !isActive {
				_ = s.DeactivateUserByAdmin(ctx, actorID, userID)
			}
		}

		report.SuccessRows++
	}

	return report, nil
}

func (s *Service) resolveEnrollmentFromNames(ctx context.Context, schoolName, className, gradeLevel string) (int64, int64, error) {
	schoolName = strings.TrimSpace(schoolName)
	className = strings.TrimSpace(className)
	gradeLevel = strings.TrimSpace(gradeLevel)

	var schoolID int64
	if err := s.db.QueryRowContext(ctx, `
		SELECT id
		FROM schools
		WHERE LOWER(name) = LOWER($1)
		  AND is_active = TRUE
		LIMIT 1
	`, schoolName).Scan(&schoolID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, 0, fmt.Errorf("school_name tidak ditemukan: %s", schoolName)
		}
		return 0, 0, fmt.Errorf("lookup school_name gagal: %w", err)
	}

	var classID int64
	if err := s.db.QueryRowContext(ctx, `
		SELECT id
		FROM classes
		WHERE school_id = $1
		  AND LOWER(name) = LOWER($2)
		  AND LOWER(grade_level) = LOWER($3)
		  AND is_active = TRUE
		LIMIT 1
	`, schoolID, className, gradeLevel).Scan(&classID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, 0, fmt.Errorf("kelas tidak ditemukan: %s | %s | %s", schoolName, gradeLevel, className)
		}
		return 0, 0, fmt.Errorf("lookup kelas gagal: %w", err)
	}
	return schoolID, classID, nil
}

func parseBoolLoose(v string) bool {
	v = strings.TrimSpace(strings.ToLower(v))
	if v == "" {
		return true
	}
	switch v {
	case "1", "true", "ya", "yes", "aktif":
		return true
	case "0", "false", "tidak", "no", "nonaktif":
		return false
	default:
		if n, err := strconv.Atoi(v); err == nil {
			return n != 0
		}
		return true
	}
}
