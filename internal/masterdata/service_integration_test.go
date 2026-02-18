package masterdata

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	internaldb "cbtlms/internal/db"
)

func TestImportStudentsCSV_DBIntegration_AllValid(t *testing.T) {
	if os.Getenv("CBTLMS_INTEGRATION") != "1" {
		t.Skip("set CBTLMS_INTEGRATION=1 to run integration tests")
	}

	dbConn := openIntegrationDB(t)
	defer dbConn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	actorID := mustActorID(ctx, t, dbConn)
	svc := NewService(dbConn)

	suffix := time.Now().UnixNano()
	u1 := fmt.Sprintf("it_mdsiswa_%d_1", suffix)
	u2 := fmt.Sprintf("it_mdsiswa_%d_2", suffix)
	schoolName := fmt.Sprintf("ITEST School %d", suffix)
	className := "X-IT-1"
	grade := "10"

	csvBody := fmt.Sprintf(`full_name,username,password,email,nisn,school_name,class_name,grade_level
Siswa Integrasi 1,%s,Password123!,%s@example.test,10001,%s,%s,%s
Siswa Integrasi 2,%s,Password123!,%s@example.test,10002,%s,%s,%s
`, u1, u1, schoolName, className, grade, u2, u2, schoolName, className, grade)

	report, err := svc.ImportStudentsCSV(ctx, actorID, strings.NewReader(csvBody))
	if err != nil {
		t.Fatalf("import csv: %v", err)
	}

	if report.TotalRows != 2 || report.SuccessRows != 2 || report.FailedRows != 0 {
		t.Fatalf("unexpected report: %+v", report)
	}

	assertUserExists(ctx, t, dbConn, u1, true)
	assertUserExists(ctx, t, dbConn, u2, true)

	cleanupUsersByUsernames(ctx, t, dbConn, []string{u1, u2})
	cleanupSchoolByName(ctx, t, dbConn, schoolName)
}

func TestImportStudentsCSV_DBIntegration_PartialInvalidRows(t *testing.T) {
	if os.Getenv("CBTLMS_INTEGRATION") != "1" {
		t.Skip("set CBTLMS_INTEGRATION=1 to run integration tests")
	}

	dbConn := openIntegrationDB(t)
	defer dbConn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	actorID := mustActorID(ctx, t, dbConn)
	svc := NewService(dbConn)

	suffix := time.Now().UnixNano()
	uValid := fmt.Sprintf("it_mdvalid_%d", suffix)
	uBadPwd := fmt.Sprintf("it_mdbadpwd_%d", suffix)
	uBadMail := fmt.Sprintf("it_mdbadmail_%d", suffix)
	schoolName := fmt.Sprintf("ITEST School Partial %d", suffix)
	className := "XI-IT-1"
	grade := "11"

	csvBody := fmt.Sprintf(`full_name,username,password,email,nisn,school_name,class_name,grade_level
Valid Student,%s,Password123!,%s@example.test,20001,%s,%s,%s
Bad Password,%s,123,%s@example.test,20002,%s,%s,%s
Bad Email,%s,Password123!,not-an-email,20003,%s,%s,%s
`, uValid, uValid, schoolName, className, grade, uBadPwd, uBadPwd, schoolName, className, grade, uBadMail, schoolName, className, grade)

	report, err := svc.ImportStudentsCSV(ctx, actorID, strings.NewReader(csvBody))
	if err != nil {
		t.Fatalf("import csv: %v", err)
	}

	if report.TotalRows != 3 || report.SuccessRows != 1 || report.FailedRows != 2 {
		t.Fatalf("unexpected report: %+v", report)
	}
	if len(report.Errors) != 2 {
		t.Fatalf("expected 2 row errors, got %+v", report.Errors)
	}

	assertUserExists(ctx, t, dbConn, uValid, true)
	assertUserExists(ctx, t, dbConn, uBadPwd, false)
	assertUserExists(ctx, t, dbConn, uBadMail, false)

	cleanupUsersByUsernames(ctx, t, dbConn, []string{uValid, uBadPwd, uBadMail})
	cleanupSchoolByName(ctx, t, dbConn, schoolName)
}

func openIntegrationDB(t *testing.T) *sql.DB {
	t.Helper()
	dsn := os.Getenv("CBTLMS_TEST_DSN")
	if strings.TrimSpace(dsn) == "" {
		dsn = "postgres://cbtlms:cbtlms_dev_password@localhost:5432/cbtlms?sslmode=disable"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	dbConn, err := internaldb.OpenPostgres(ctx, dsn)
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	return dbConn
}

func mustActorID(ctx context.Context, t *testing.T, db *sql.DB) int64 {
	t.Helper()
	var actorID int64
	err := db.QueryRowContext(ctx, `SELECT id FROM users WHERE username='admin' LIMIT 1`).Scan(&actorID)
	if err != nil {
		t.Fatalf("load admin user: %v", err)
	}
	return actorID
}

func assertUserExists(ctx context.Context, t *testing.T, db *sql.DB, username string, expect bool) {
	t.Helper()
	var exists bool
	err := db.QueryRowContext(ctx, `
		SELECT EXISTS(SELECT 1 FROM users WHERE username = $1)
	`, username).Scan(&exists)
	if err != nil {
		t.Fatalf("check user existence: %v", err)
	}
	if exists != expect {
		t.Fatalf("user existence mismatch username=%s got=%v expect=%v", username, exists, expect)
	}
}

func cleanupUsersByUsernames(ctx context.Context, t *testing.T, db *sql.DB, usernames []string) {
	t.Helper()
	for _, username := range usernames {
		if strings.TrimSpace(username) == "" {
			continue
		}
		_, _ = db.ExecContext(ctx, `DELETE FROM users WHERE username = $1`, username)
	}
}

func cleanupSchoolByName(ctx context.Context, t *testing.T, db *sql.DB, schoolName string) {
	t.Helper()
	_, _ = db.ExecContext(ctx, `DELETE FROM schools WHERE name = $1`, schoolName)
}
