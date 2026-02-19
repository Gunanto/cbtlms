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
	headers := []string{"username", "email", "full_name", "role", "school_name", "is_active", "account_status", "created_at"}
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
		values := []any{
			it.Username,
			email,
			it.FullName,
			it.Role,
			schoolName,
			it.IsActive,
			it.AccountStatus,
			it.CreatedAt.Format("2006-01-02 15:04:05"),
		}
		for col, v := range values {
			cell, _ := excelize.CoordinatesToCellName(col+1, row)
			_ = f.SetCellValue(sheet, cell, v)
		}
	}
	_ = f.SetColWidth(sheet, "A", "H", 22)

	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		return nil, fmt.Errorf("write excel: %w", err)
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
