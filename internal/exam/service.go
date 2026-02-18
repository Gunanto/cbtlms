package exam

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
)

var (
	ErrExamNotFound       = errors.New("exam not found")
	ErrAttemptNotFound    = errors.New("attempt not found")
	ErrAttemptNotEditable = errors.New("attempt is not editable")
	ErrQuestionNotInExam  = errors.New("question not in exam")
	ErrAttemptForbidden   = errors.New("attempt forbidden")
	ErrAttemptNotFinal    = errors.New("attempt not final")
	ErrResultPolicyDenied = errors.New("result not available by review policy")
)

type Service struct {
	db                 *sql.DB
	defaultExamMinutes int
}

type Attempt struct {
	ID        int64     `json:"id"`
	ExamID    int64     `json:"exam_id"`
	StudentID int64     `json:"student_id"`
	Status    string    `json:"status"`
	StartedAt time.Time `json:"started_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

type AttemptSummary struct {
	ID              int64      `json:"id"`
	ExamID          int64      `json:"exam_id"`
	StudentID       int64      `json:"student_id"`
	Status          string     `json:"status"`
	StartedAt       time.Time  `json:"started_at"`
	ExpiresAt       time.Time  `json:"expires_at"`
	SubmittedAt     *time.Time `json:"submitted_at,omitempty"`
	RemainingSecs   int64      `json:"remaining_secs"`
	TotalQuestions  int        `json:"total_questions"`
	Answered        int        `json:"answered"`
	Doubt           int        `json:"doubt"`
	TotalCorrect    int        `json:"total_correct"`
	TotalWrong      int        `json:"total_wrong"`
	TotalUnanswered int        `json:"total_unanswered"`
	Score           float64    `json:"score"`
}

type SaveAnswerInput struct {
	AttemptID     int64
	QuestionID    int64
	AnswerPayload json.RawMessage
	IsDoubt       bool
}

type AttemptResult struct {
	Summary AttemptSummary      `json:"summary"`
	Items   []AttemptResultItem `json:"items"`
}

type AttemptResultItem struct {
	QuestionID int64    `json:"question_id"`
	Selected   []string `json:"selected"`
	Correct    []string `json:"correct"`
	IsCorrect  *bool    `json:"is_correct,omitempty"`
	EarnedScore float64 `json:"earned_score"`
}

type attemptRow struct {
	ID              int64
	ExamID          int64
	StudentID       int64
	Status          string
	StartedAt       time.Time
	ExpiresAt       time.Time
	SubmittedAt     sql.NullTime
	Score           sql.NullFloat64
	TotalCorrect    sql.NullInt64
	TotalWrong      sql.NullInt64
	TotalUnanswered sql.NullInt64
}

type questionEvalRow struct {
	QuestionID  int64
	Weight      float64
	Payload     []byte
	CorrectKeys []string
}

func NewService(db *sql.DB, defaultExamMinutes int) *Service {
	if defaultExamMinutes <= 0 {
		defaultExamMinutes = 90
	}
	return &Service{db: db, defaultExamMinutes: defaultExamMinutes}
}

func (s *Service) StartAttempt(ctx context.Context, examID, studentID int64) (*Attempt, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, exam_id, student_id, status, started_at, expires_at
		FROM attempts
		WHERE exam_id = $1 AND student_id = $2
	`, examID, studentID)

	var existing Attempt
	if err := row.Scan(&existing.ID, &existing.ExamID, &existing.StudentID, &existing.Status, &existing.StartedAt, &existing.ExpiresAt); err == nil {
		return &existing, nil
	} else if !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("query existing attempt: %w", err)
	}

	duration := s.defaultExamMinutes
	if err := s.db.QueryRowContext(ctx, `
		SELECT duration_minutes
		FROM exams
		WHERE id = $1 AND is_active = TRUE
	`, examID).Scan(&duration); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrExamNotFound
		}
		return nil, fmt.Errorf("query exam duration: %w", err)
	}

	row = s.db.QueryRowContext(ctx, `
		INSERT INTO attempts (
			exam_id,
			student_id,
			status,
			started_at,
			expires_at
		) VALUES (
			$1,
			$2,
			'in_progress',
			now(),
			now() + make_interval(mins => $3)
		)
		RETURNING id, exam_id, student_id, status, started_at, expires_at
	`, examID, studentID, duration)

	var created Attempt
	if err := row.Scan(&created.ID, &created.ExamID, &created.StudentID, &created.Status, &created.StartedAt, &created.ExpiresAt); err != nil {
		return nil, fmt.Errorf("insert attempt: %w", err)
	}

	return &created, nil
}

func (s *Service) GetAttemptSummary(ctx context.Context, attemptID int64) (*AttemptSummary, error) {
	row, err := s.loadAttemptRow(ctx, s.db, attemptID)
	if err != nil {
		return nil, err
	}

	if row.Status == "in_progress" && time.Now().After(row.ExpiresAt) {
		if _, err := s.finalizeAttempt(ctx, attemptID, "expired"); err != nil {
			return nil, err
		}
		row, err = s.loadAttemptRow(ctx, s.db, attemptID)
		if err != nil {
			return nil, err
		}
	}

	return s.buildSummaryFromRow(ctx, s.db, row)
}

func (s *Service) SaveAnswer(ctx context.Context, input SaveAnswerInput) error {
	row, err := s.loadAttemptRow(ctx, s.db, input.AttemptID)
	if err != nil {
		return err
	}

	if row.Status != "in_progress" {
		return ErrAttemptNotEditable
	}
	if time.Now().After(row.ExpiresAt) {
		_, _ = s.finalizeAttempt(ctx, input.AttemptID, "expired")
		return ErrAttemptNotEditable
	}

	var exists bool
	if err := s.db.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM exam_questions eq
			WHERE eq.exam_id = $1 AND eq.question_id = $2
		)
	`, row.ExamID, input.QuestionID).Scan(&exists); err != nil {
		return fmt.Errorf("validate question in exam: %w", err)
	}
	if !exists {
		return ErrQuestionNotInExam
	}

	if len(input.AnswerPayload) == 0 {
		input.AnswerPayload = json.RawMessage(`{}`)
	}

	_, err = s.db.ExecContext(ctx, `
		INSERT INTO attempt_answers (
			attempt_id,
			question_id,
			answer_payload,
			is_doubt,
			is_final,
			updated_at
		) VALUES ($1, $2, $3::jsonb, $4, FALSE, now())
		ON CONFLICT (attempt_id, question_id)
		DO UPDATE SET
			answer_payload = EXCLUDED.answer_payload,
			is_doubt = EXCLUDED.is_doubt,
			updated_at = now()
	`, input.AttemptID, input.QuestionID, []byte(input.AnswerPayload), input.IsDoubt)
	if err != nil {
		return fmt.Errorf("upsert answer: %w", err)
	}

	return nil
}

func (s *Service) SubmitAttempt(ctx context.Context, attemptID int64) (*AttemptSummary, error) {
	return s.finalizeAttempt(ctx, attemptID, "submitted")
}

func (s *Service) GetAttemptOwner(ctx context.Context, attemptID int64) (int64, error) {
	var studentID int64
	if err := s.db.QueryRowContext(ctx, `
		SELECT student_id
		FROM attempts
		WHERE id = $1
	`, attemptID).Scan(&studentID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, ErrAttemptNotFound
		}
		return 0, fmt.Errorf("load attempt owner: %w", err)
	}
	return studentID, nil
}

func (s *Service) GetAttemptResult(ctx context.Context, attemptID int64) (*AttemptResult, error) {
	row, err := s.loadAttemptRow(ctx, s.db, attemptID)
	if err != nil {
		return nil, err
	}

	if row.Status == "in_progress" {
		if time.Now().After(row.ExpiresAt) {
			if _, err := s.finalizeAttempt(ctx, attemptID, "expired"); err != nil {
				return nil, err
			}
			row, err = s.loadAttemptRow(ctx, s.db, attemptID)
			if err != nil {
				return nil, err
			}
		} else {
			return nil, ErrAttemptNotFinal
		}
	}

	reviewPolicy, examEndAt, err := s.loadExamReviewPolicy(ctx, row.ExamID)
	if err != nil {
		return nil, err
	}
	if !canViewResultByPolicy(reviewPolicy, row.Status, examEndAt) {
		return nil, ErrResultPolicyDenied
	}

	summary, err := s.buildSummaryFromRow(ctx, s.db, row)
	if err != nil {
		return nil, err
	}

	items, err := s.loadAttemptResultItems(ctx, attemptID)
	if err != nil {
		return nil, err
	}

	return &AttemptResult{
		Summary: *summary,
		Items:   items,
	}, nil
}

func (s *Service) loadExamReviewPolicy(ctx context.Context, examID int64) (string, *time.Time, error) {
	var policy sql.NullString
	var endAt sql.NullTime
	if err := s.db.QueryRowContext(ctx, `
		SELECT review_policy, end_at
		FROM exams
		WHERE id = $1
	`, examID).Scan(&policy, &endAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", nil, ErrExamNotFound
		}
		return "", nil, fmt.Errorf("load exam review policy: %w", err)
	}

	p := strings.TrimSpace(policy.String)
	if p == "" {
		p = "after_submit"
	}

	var e *time.Time
	if endAt.Valid {
		e = &endAt.Time
	}
	return p, e, nil
}

func (s *Service) finalizeAttempt(ctx context.Context, attemptID int64, finalStatus string) (*AttemptSummary, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin finalize tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	row, err := s.loadAttemptRowForUpdate(ctx, tx, attemptID)
	if err != nil {
		return nil, err
	}

	if row.Status != "in_progress" {
		summary, err := s.buildSummaryFromRow(ctx, tx, row)
		if err != nil {
			return nil, err
		}
		if err := tx.Commit(); err != nil {
			return nil, fmt.Errorf("commit finalize existing: %w", err)
		}
		return summary, nil
	}

	if finalStatus == "expired" && time.Now().Before(row.ExpiresAt) {
		summary, err := s.buildSummaryFromRow(ctx, tx, row)
		if err != nil {
			return nil, err
		}
		if err := tx.Commit(); err != nil {
			return nil, fmt.Errorf("commit finalize noop: %w", err)
		}
		return summary, nil
	}

	evals, err := s.loadQuestionEvaluations(ctx, tx, row.ExamID, row.ID)
	if err != nil {
		return nil, err
	}

	if _, err := tx.ExecContext(ctx, `DELETE FROM attempt_scores WHERE attempt_id = $1`, row.ID); err != nil {
		return nil, fmt.Errorf("clear attempt_scores: %w", err)
	}

	totalQuestions := len(evals)
	answered := 0
	totalCorrect := 0
	score := 0.0

	for _, ev := range evals {
		selected := extractSelectedKeys(ev.Payload)
		isAnswered := len(selected) > 0
		if isAnswered {
			answered++
		}

		isCorrect := false
		earned := 0.0
		if isAnswered && len(ev.CorrectKeys) > 0 && equalSet(selected, ev.CorrectKeys) {
			isCorrect = true
			earned = ev.Weight
			totalCorrect++
			score += ev.Weight
		}

		var isCorrectPtr interface{}
		if isAnswered {
			isCorrectPtr = isCorrect
		} else {
			isCorrectPtr = nil
		}

		feedback := map[string]interface{}{
			"selected": selected,
			"correct":  ev.CorrectKeys,
		}
		feedbackJSON, _ := json.Marshal(feedback)

		if _, err := tx.ExecContext(ctx, `
			INSERT INTO attempt_scores (
				attempt_id,
				question_id,
				earned_score,
				is_correct,
				feedback
			) VALUES ($1,$2,$3,$4,$5::jsonb)
		`, row.ID, ev.QuestionID, earned, isCorrectPtr, feedbackJSON); err != nil {
			return nil, fmt.Errorf("insert attempt_score: %w", err)
		}
	}

	totalWrong := answered - totalCorrect
	totalUnanswered := totalQuestions - answered

	if _, err := tx.ExecContext(ctx, `
		UPDATE attempt_answers
		SET is_final = TRUE,
			updated_at = now()
		WHERE attempt_id = $1
	`, row.ID); err != nil {
		return nil, fmt.Errorf("mark attempt_answers final: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `
		UPDATE attempts
		SET status = $2,
			submitted_at = now(),
			score = $3,
			total_correct = $4,
			total_wrong = $5,
			total_unanswered = $6
		WHERE id = $1
	`, row.ID, finalStatus, score, totalCorrect, totalWrong, totalUnanswered); err != nil {
		return nil, fmt.Errorf("update attempt final: %w", err)
	}

	row, err = s.loadAttemptRowForUpdate(ctx, tx, row.ID)
	if err != nil {
		return nil, err
	}

	summary, err := s.buildSummaryFromRow(ctx, tx, row)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit finalize: %w", err)
	}
	return summary, nil
}

func (s *Service) loadQuestionEvaluations(ctx context.Context, q queryable, examID, attemptID int64) ([]questionEvalRow, error) {
	rows, err := q.QueryContext(ctx, `
		SELECT
			eq.question_id,
			eq.weight,
			COALESCE(aa.answer_payload, '{}'::jsonb) AS answer_payload,
			COALESCE(
				json_agg(qo.option_key) FILTER (WHERE qo.is_correct),
				'[]'::json
			) AS correct_keys_json
		FROM exam_questions eq
		LEFT JOIN attempt_answers aa
			ON aa.attempt_id = $1
			AND aa.question_id = eq.question_id
		LEFT JOIN question_options qo
			ON qo.question_id = eq.question_id
		WHERE eq.exam_id = $2
		GROUP BY eq.question_id, eq.weight, aa.answer_payload, eq.seq_no
		ORDER BY eq.seq_no
	`, attemptID, examID)
	if err != nil {
		return nil, fmt.Errorf("query evaluations: %w", err)
	}
	defer rows.Close()

	out := make([]questionEvalRow, 0)
	for rows.Next() {
		var r questionEvalRow
		var correctKeysJSON []byte
		if err := rows.Scan(&r.QuestionID, &r.Weight, &r.Payload, &correctKeysJSON); err != nil {
			return nil, fmt.Errorf("scan evaluation row: %w", err)
		}
		if len(correctKeysJSON) > 0 {
			if err := json.Unmarshal(correctKeysJSON, &r.CorrectKeys); err != nil {
				return nil, fmt.Errorf("decode correct keys json: %w", err)
			}
		}
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate evaluations: %w", err)
	}
	return out, nil
}

func (s *Service) loadAttemptResultItems(ctx context.Context, attemptID int64) ([]AttemptResultItem, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT
			question_id,
			earned_score,
			is_correct,
			feedback
		FROM attempt_scores
		WHERE attempt_id = $1
		ORDER BY question_id ASC
	`, attemptID)
	if err != nil {
		return nil, fmt.Errorf("query attempt result items: %w", err)
	}
	defer rows.Close()

	items := make([]AttemptResultItem, 0)
	for rows.Next() {
		var (
			item        AttemptResultItem
			isCorrect   sql.NullBool
			feedbackRaw []byte
		)
		if err := rows.Scan(&item.QuestionID, &item.EarnedScore, &isCorrect, &feedbackRaw); err != nil {
			return nil, fmt.Errorf("scan result item: %w", err)
		}
		if isCorrect.Valid {
			v := isCorrect.Bool
			item.IsCorrect = &v
		}
		if len(feedbackRaw) > 0 {
			var f map[string]interface{}
			if err := json.Unmarshal(feedbackRaw, &f); err == nil {
				item.Selected = anyToStringSlice(f["selected"])
				item.Correct = anyToStringSlice(f["correct"])
			}
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate result items: %w", err)
	}
	return items, nil
}

func (s *Service) buildSummaryFromRow(ctx context.Context, q queryable, row *attemptRow) (*AttemptSummary, error) {
	totalQuestions, answered, doubt, err := s.countProgress(ctx, q, row.ID, row.ExamID)
	if err != nil {
		return nil, err
	}

	summary := &AttemptSummary{
		ID:             row.ID,
		ExamID:         row.ExamID,
		StudentID:      row.StudentID,
		Status:         row.Status,
		StartedAt:      row.StartedAt,
		ExpiresAt:      row.ExpiresAt,
		TotalQuestions: totalQuestions,
		Answered:       answered,
		Doubt:          doubt,
		RemainingSecs:  remainingSeconds(row.Status, row.ExpiresAt),
	}

	if row.SubmittedAt.Valid {
		summary.SubmittedAt = &row.SubmittedAt.Time
	}
	if row.Score.Valid {
		summary.Score = row.Score.Float64
	}
	if row.TotalCorrect.Valid {
		summary.TotalCorrect = int(row.TotalCorrect.Int64)
	}
	if row.TotalWrong.Valid {
		summary.TotalWrong = int(row.TotalWrong.Int64)
	}
	if row.TotalUnanswered.Valid {
		summary.TotalUnanswered = int(row.TotalUnanswered.Int64)
	} else {
		summary.TotalUnanswered = totalQuestions - answered
	}

	if row.Status == "in_progress" {
		summary.TotalCorrect = 0
		summary.TotalWrong = 0
		summary.TotalUnanswered = totalQuestions - answered
	}

	return summary, nil
}

func (s *Service) countProgress(ctx context.Context, q queryable, attemptID, examID int64) (totalQuestions, answered, doubt int, err error) {
	if err = q.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM exam_questions
		WHERE exam_id = $1
	`, examID).Scan(&totalQuestions); err != nil {
		return 0, 0, 0, fmt.Errorf("count total questions: %w", err)
	}

	if err = q.QueryRowContext(ctx, `
		SELECT
			COUNT(*) FILTER (
				WHERE answer_payload IS NOT NULL
				  AND answer_payload <> '{}'::jsonb
			),
			COUNT(*) FILTER (WHERE is_doubt = TRUE)
		FROM attempt_answers
		WHERE attempt_id = $1
	`, attemptID).Scan(&answered, &doubt); err != nil {
		return 0, 0, 0, fmt.Errorf("count answers: %w", err)
	}

	if answered > totalQuestions {
		answered = totalQuestions
	}

	return totalQuestions, answered, doubt, nil
}

func (s *Service) loadAttemptRow(ctx context.Context, q queryable, attemptID int64) (*attemptRow, error) {
	row := &attemptRow{}
	err := q.QueryRowContext(ctx, `
		SELECT
			id,
			exam_id,
			student_id,
			status,
			started_at,
			expires_at,
			submitted_at,
			score,
			total_correct,
			total_wrong,
			total_unanswered
		FROM attempts
		WHERE id = $1
	`, attemptID).Scan(
		&row.ID,
		&row.ExamID,
		&row.StudentID,
		&row.Status,
		&row.StartedAt,
		&row.ExpiresAt,
		&row.SubmittedAt,
		&row.Score,
		&row.TotalCorrect,
		&row.TotalWrong,
		&row.TotalUnanswered,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrAttemptNotFound
		}
		return nil, fmt.Errorf("load attempt: %w", err)
	}
	return row, nil
}

func (s *Service) loadAttemptRowForUpdate(ctx context.Context, tx *sql.Tx, attemptID int64) (*attemptRow, error) {
	row := &attemptRow{}
	err := tx.QueryRowContext(ctx, `
		SELECT
			id,
			exam_id,
			student_id,
			status,
			started_at,
			expires_at,
			submitted_at,
			score,
			total_correct,
			total_wrong,
			total_unanswered
		FROM attempts
		WHERE id = $1
		FOR UPDATE
	`, attemptID).Scan(
		&row.ID,
		&row.ExamID,
		&row.StudentID,
		&row.Status,
		&row.StartedAt,
		&row.ExpiresAt,
		&row.SubmittedAt,
		&row.Score,
		&row.TotalCorrect,
		&row.TotalWrong,
		&row.TotalUnanswered,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrAttemptNotFound
		}
		return nil, fmt.Errorf("load attempt for update: %w", err)
	}
	return row, nil
}

type queryable interface {
	QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
}

func remainingSeconds(status string, expiresAt time.Time) int64 {
	if status != "in_progress" {
		return 0
	}
	remaining := time.Until(expiresAt)
	if remaining <= 0 {
		return 0
	}
	return int64(remaining.Seconds())
}

func extractSelectedKeys(payload []byte) []string {
	if len(payload) == 0 {
		return nil
	}

	var obj map[string]interface{}
	if err := json.Unmarshal(payload, &obj); err != nil {
		return nil
	}

	v, ok := obj["selected"]
	if !ok {
		return nil
	}

	set := map[string]struct{}{}
	switch t := v.(type) {
	case string:
		x := strings.TrimSpace(t)
		if x != "" {
			set[x] = struct{}{}
		}
	case []interface{}:
		for _, item := range t {
			s, ok := item.(string)
			if !ok {
				continue
			}
			s = strings.TrimSpace(s)
			if s != "" {
				set[s] = struct{}{}
			}
		}
	}

	out := make([]string, 0, len(set))
	for k := range set {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func equalSet(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	aa := append([]string(nil), a...)
	bb := append([]string(nil), b...)
	sort.Strings(aa)
	sort.Strings(bb)
	for i := range aa {
		if aa[i] != bb[i] {
			return false
		}
	}
	return true
}

func anyToStringSlice(v interface{}) []string {
	set := map[string]struct{}{}
	switch t := v.(type) {
	case string:
		s := strings.TrimSpace(t)
		if s != "" {
			set[s] = struct{}{}
		}
	case []interface{}:
		for _, it := range t {
			s, ok := it.(string)
			if !ok {
				continue
			}
			s = strings.TrimSpace(s)
			if s != "" {
				set[s] = struct{}{}
			}
		}
	case []string:
		for _, s := range t {
			s = strings.TrimSpace(s)
			if s != "" {
				set[s] = struct{}{}
			}
		}
	}
	out := make([]string, 0, len(set))
	for k := range set {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func canViewResultByPolicy(policy, attemptStatus string, examEndAt *time.Time) bool {
	finalized := attemptStatus == "submitted" || attemptStatus == "expired"
	if !finalized {
		return false
	}

	switch strings.TrimSpace(policy) {
	case "", "after_submit":
		return true
	case "after_exam_end":
		if examEndAt == nil {
			return false
		}
		return time.Now().After(*examEndAt) || time.Now().Equal(*examEndAt)
	case "immediate":
		return true
	case "disabled":
		return false
	default:
		// Safe fallback: treat unknown policy like after_submit.
		return true
	}
}
