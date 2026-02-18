package exam

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

func TestSubmitAttemptIdempotent_DBIntegration(t *testing.T) {
	if os.Getenv("CBTLMS_INTEGRATION") != "1" {
		t.Skip("set CBTLMS_INTEGRATION=1 to run integration tests")
	}

	dsn := os.Getenv("CBTLMS_TEST_DSN")
	if strings.TrimSpace(dsn) == "" {
		dsn = "postgres://cbtlms:cbtlms_dev_password@localhost:5432/cbtlms?sslmode=disable"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	dbConn, err := internaldb.OpenPostgres(ctx, dsn)
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	defer dbConn.Close()

	svc := NewService(dbConn, 90)

	suffix := time.Now().UnixNano()
	subjectName := fmt.Sprintf("ITEST Subject %d", suffix)
	examCode := fmt.Sprintf("ITEST-EXAM-%d", suffix)
	studentUsername := fmt.Sprintf("itest_student_%d", suffix)

	tx, err := dbConn.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("begin seed tx: %v", err)
	}
	defer func() { _ = tx.Rollback() }()

	var subjectID int64
	err = tx.QueryRowContext(ctx, `
		INSERT INTO subjects (education_level, subject_type, name, is_active)
		VALUES ('SMA', 'Wajib', $1, TRUE)
		RETURNING id
	`, subjectName).Scan(&subjectID)
	if err != nil {
		t.Fatalf("insert subject: %v", err)
	}

	var studentID int64
	err = tx.QueryRowContext(ctx, `
		INSERT INTO users (
			username, password_hash, full_name, role, is_active,
			email, email_verified_at, account_status, created_at, updated_at
		) VALUES (
			$1, 'dummy_hash', 'Integration Student', 'siswa', TRUE,
			$2, now(), 'active', now(), now()
		)
		RETURNING id
	`, studentUsername, studentUsername+"@example.test").Scan(&studentID)
	if err != nil {
		t.Fatalf("insert student: %v", err)
	}

	var examID int64
	err = tx.QueryRowContext(ctx, `
		INSERT INTO exams (
			code, title, subject_id, duration_minutes,
			randomize_questions, randomize_options, review_policy,
			is_active, created_at
		) VALUES (
			$1, 'Integration Exam', $2, 60,
			FALSE, FALSE, 'after_submit',
			TRUE, now()
		)
		RETURNING id
	`, examCode, subjectID).Scan(&examID)
	if err != nil {
		t.Fatalf("insert exam: %v", err)
	}

	var questionID int64
	err = tx.QueryRowContext(ctx, `
		INSERT INTO questions (
			subject_id, question_type, stem_html, metadata,
			is_active, version, created_at, updated_at
		) VALUES (
			$1, 'pg_tunggal', '<p>2+2=?</p>', '{}'::jsonb,
			TRUE, 1, now(), now()
		)
		RETURNING id
	`, subjectID).Scan(&questionID)
	if err != nil {
		t.Fatalf("insert question: %v", err)
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO question_options (question_id, option_key, option_html, is_correct)
		VALUES
		($1, 'A', '<p>3</p>', FALSE),
		($1, 'B', '<p>4</p>', TRUE),
		($1, 'C', '<p>5</p>', FALSE)
	`, questionID)
	if err != nil {
		t.Fatalf("insert options: %v", err)
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO exam_questions (exam_id, question_id, seq_no, weight)
		VALUES ($1, $2, 1, 2.0)
	`, examID, questionID)
	if err != nil {
		t.Fatalf("insert exam_question: %v", err)
	}

	var attemptID int64
	err = tx.QueryRowContext(ctx, `
		INSERT INTO attempts (
			exam_id, student_id, status, started_at, expires_at
		) VALUES (
			$1, $2, 'in_progress', now() - interval '5 minute', now() + interval '30 minute'
		)
		RETURNING id
	`, examID, studentID).Scan(&attemptID)
	if err != nil {
		t.Fatalf("insert attempt: %v", err)
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO attempt_answers (attempt_id, question_id, answer_payload, is_doubt, is_final, updated_at)
		VALUES ($1, $2, '{"selected":["B"]}'::jsonb, FALSE, FALSE, now())
	`, attemptID, questionID)
	if err != nil {
		t.Fatalf("insert attempt_answer: %v", err)
	}

	if err := tx.Commit(); err != nil {
		t.Fatalf("commit seed: %v", err)
	}

	defer cleanupIntegrationFixture(t, dbConn, examID, questionID, subjectID, studentID)

	first, err := svc.SubmitAttempt(ctx, attemptID)
	if err != nil {
		t.Fatalf("first submit: %v", err)
	}
	second, err := svc.SubmitAttempt(ctx, attemptID)
	if err != nil {
		t.Fatalf("second submit: %v", err)
	}

	if first.Status != "submitted" || second.Status != "submitted" {
		t.Fatalf("expected submitted status, got first=%s second=%s", first.Status, second.Status)
	}
	if first.Score != second.Score {
		t.Fatalf("score changed across submits: first=%v second=%v", first.Score, second.Score)
	}
	if first.TotalCorrect != second.TotalCorrect || first.TotalWrong != second.TotalWrong || first.TotalUnanswered != second.TotalUnanswered {
		t.Fatalf("summary counts changed across submits")
	}

	if first.SubmittedAt == nil || second.SubmittedAt == nil {
		t.Fatalf("submitted_at should be set")
	}
	if !first.SubmittedAt.Equal(*second.SubmittedAt) {
		t.Fatalf("submitted_at changed across idempotent submit: first=%v second=%v", first.SubmittedAt, second.SubmittedAt)
	}

	var scoreRows int
	err = dbConn.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM attempt_scores WHERE attempt_id = $1
	`, attemptID).Scan(&scoreRows)
	if err != nil {
		t.Fatalf("count attempt_scores: %v", err)
	}
	if scoreRows != 1 {
		t.Fatalf("expected exactly 1 attempt_scores row, got %d", scoreRows)
	}

	var storedStatus string
	var storedScore float64
	err = dbConn.QueryRowContext(ctx, `
		SELECT status, score FROM attempts WHERE id = $1
	`, attemptID).Scan(&storedStatus, &storedScore)
	if err != nil {
		t.Fatalf("load finalized attempt: %v", err)
	}
	if storedStatus != "submitted" {
		t.Fatalf("expected DB status submitted, got %s", storedStatus)
	}
	if storedScore != first.Score {
		t.Fatalf("expected DB score %v, got %v", first.Score, storedScore)
	}
}

func cleanupIntegrationFixture(t *testing.T, db *sql.DB, examID, questionID, subjectID, studentID int64) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Logf("cleanup begin tx failed: %v", err)
		return
	}
	defer func() { _ = tx.Rollback() }()

	_, _ = tx.ExecContext(ctx, `DELETE FROM attempt_scores WHERE question_id = $1`, questionID)
	_, _ = tx.ExecContext(ctx, `DELETE FROM attempt_answers WHERE question_id = $1`, questionID)
	_, _ = tx.ExecContext(ctx, `DELETE FROM attempts WHERE exam_id = $1`, examID)
	_, _ = tx.ExecContext(ctx, `DELETE FROM exam_questions WHERE exam_id = $1`, examID)
	_, _ = tx.ExecContext(ctx, `DELETE FROM exams WHERE id = $1`, examID)
	_, _ = tx.ExecContext(ctx, `DELETE FROM question_options WHERE question_id = $1`, questionID)
	_, _ = tx.ExecContext(ctx, `DELETE FROM question_versions WHERE question_id = $1`, questionID)
	_, _ = tx.ExecContext(ctx, `DELETE FROM questions WHERE id = $1`, questionID)
	_, _ = tx.ExecContext(ctx, `DELETE FROM stimuli WHERE subject_id = $1`, subjectID)
	_, _ = tx.ExecContext(ctx, `DELETE FROM subjects WHERE id = $1`, subjectID)
	_, _ = tx.ExecContext(ctx, `DELETE FROM auth_identities WHERE user_id = $1`, studentID)
	_, _ = tx.ExecContext(ctx, `DELETE FROM auth_sessions WHERE user_id = $1`, studentID)
	_, _ = tx.ExecContext(ctx, `DELETE FROM student_profiles WHERE user_id = $1`, studentID)
	_, _ = tx.ExecContext(ctx, `DELETE FROM users WHERE id = $1`, studentID)

	if err := tx.Commit(); err != nil {
		t.Logf("cleanup commit failed: %v", err)
	}
}
