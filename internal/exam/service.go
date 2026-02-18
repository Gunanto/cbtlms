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
	ErrInvalidInput       = errors.New("invalid input")
	ErrExamNotFound       = errors.New("exam not found")
	ErrAttemptNotFound    = errors.New("attempt not found")
	ErrAttemptNotEditable = errors.New("attempt is not editable")
	ErrQuestionNotInExam  = errors.New("question not in exam")
	ErrAttemptForbidden   = errors.New("attempt forbidden")
	ErrAttemptNotFinal    = errors.New("attempt not final")
	ErrResultPolicyDenied = errors.New("result not available by review policy")
	ErrInvalidEventType   = errors.New("invalid event type")
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

type SubjectOption struct {
	ID             int64  `json:"id"`
	EducationLevel string `json:"education_level"`
	SubjectType    string `json:"subject_type"`
	Name           string `json:"name"`
}

type ExamOption struct {
	ID           int64      `json:"id"`
	Code         string     `json:"code"`
	Title        string     `json:"title"`
	SubjectID    int64      `json:"subject_id"`
	EndAt        *time.Time `json:"end_at,omitempty"`
	ReviewPolicy string     `json:"review_policy"`
}

type AttemptResultItem struct {
	QuestionID  int64            `json:"question_id"`
	Selected    []string         `json:"selected"`
	Correct     []string         `json:"correct"`
	IsCorrect   *bool            `json:"is_correct,omitempty"`
	EarnedScore float64          `json:"earned_score"`
	Reason      string           `json:"reason,omitempty"`
	Breakdown   []StatementScore `json:"breakdown,omitempty"`
}

type AttemptQuestion struct {
	AttemptID     int64            `json:"attempt_id"`
	QuestionID    int64            `json:"question_id"`
	SeqNo         int              `json:"seq_no"`
	QuestionType  string           `json:"question_type"`
	StemHTML      string           `json:"stem_html"`
	StimulusHTML  *string          `json:"stimulus_html,omitempty"`
	AnswerKey     json.RawMessage  `json:"answer_key"`
	AnswerPayload json.RawMessage  `json:"answer_payload"`
	IsDoubt       bool             `json:"is_doubt"`
	Options       []QuestionOption `json:"options,omitempty"`
}

type QuestionOption struct {
	OptionKey  string `json:"option_key"`
	OptionHTML string `json:"option_html"`
}

type AttemptEventInput struct {
	AttemptID   int64
	EventType   string
	Payload     json.RawMessage
	ClientTS    *time.Time
	ActorUserID int64
}

type AttemptEvent struct {
	ID          int64           `json:"id"`
	AttemptID   int64           `json:"attempt_id"`
	EventType   string          `json:"event_type"`
	Payload     json.RawMessage `json:"payload"`
	ClientTS    *time.Time      `json:"client_ts,omitempty"`
	ServerTS    time.Time       `json:"server_ts"`
	ActorUserID *int64          `json:"actor_user_id,omitempty"`
	CreatedAt   time.Time       `json:"created_at"`
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
	QuestionID   int64
	QuestionType string
	Weight       float64
	Payload      []byte
	AnswerKey    []byte
	CorrectKeys  []string
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

func (s *Service) ListSubjects(ctx context.Context, level, subjectType string) ([]SubjectOption, error) {
	level = strings.TrimSpace(level)
	subjectType = strings.TrimSpace(subjectType)

	query := `
		SELECT id, education_level, subject_type, name
		FROM subjects
		WHERE is_active = TRUE
	`
	args := make([]any, 0, 2)
	idx := 1
	if level != "" {
		query += fmt.Sprintf(" AND education_level = $%d", idx)
		args = append(args, level)
		idx++
	}
	if subjectType != "" {
		query += fmt.Sprintf(" AND subject_type = $%d", idx)
		args = append(args, subjectType)
		idx++
	}
	query += " ORDER BY education_level, subject_type, name"

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query subjects: %w", err)
	}
	defer rows.Close()

	items := make([]SubjectOption, 0)
	for rows.Next() {
		var it SubjectOption
		if err := rows.Scan(&it.ID, &it.EducationLevel, &it.SubjectType, &it.Name); err != nil {
			return nil, fmt.Errorf("scan subject: %w", err)
		}
		items = append(items, it)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate subjects: %w", err)
	}
	return items, nil
}

func (s *Service) ListExamsBySubject(ctx context.Context, subjectID int64) ([]ExamOption, error) {
	if subjectID <= 0 {
		return nil, ErrInvalidInput
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT id, code, title, subject_id, end_at, review_policy
		FROM exams
		WHERE subject_id = $1
		  AND is_active = TRUE
		ORDER BY created_at DESC, id DESC
	`, subjectID)
	if err != nil {
		return nil, fmt.Errorf("query exams by subject: %w", err)
	}
	defer rows.Close()

	items := make([]ExamOption, 0)
	for rows.Next() {
		var it ExamOption
		var endAt sql.NullTime
		if err := rows.Scan(&it.ID, &it.Code, &it.Title, &it.SubjectID, &endAt, &it.ReviewPolicy); err != nil {
			return nil, fmt.Errorf("scan exam: %w", err)
		}
		if endAt.Valid {
			it.EndAt = &endAt.Time
		}
		items = append(items, it)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate exams: %w", err)
	}
	return items, nil
}

func (s *Service) GetAttemptQuestion(ctx context.Context, attemptID int64, questionNo int) (*AttemptQuestion, error) {
	if attemptID <= 0 || questionNo <= 0 {
		return nil, ErrInvalidInput
	}

	row, err := s.loadAttemptRow(ctx, s.db, attemptID)
	if err != nil {
		return nil, err
	}
	if row.Status == "in_progress" && time.Now().After(row.ExpiresAt) {
		if _, err := s.finalizeAttempt(ctx, attemptID, "expired"); err != nil {
			return nil, err
		}
	}

	r := s.db.QueryRowContext(ctx, `
		SELECT
			a.id,
			eq.question_id,
			eq.seq_no,
			q.question_type,
			q.stem_html,
			q.stimulus_html,
			COALESCE(qv.answer_key, '{}'::jsonb) AS answer_key,
			COALESCE(aa.answer_payload, '{}'::jsonb) AS answer_payload,
			COALESCE(aa.is_doubt, FALSE) AS is_doubt
		FROM attempts a
		JOIN exam_questions eq
			ON eq.exam_id = a.exam_id
		JOIN questions q
			ON q.id = eq.question_id
		LEFT JOIN attempt_answers aa
			ON aa.attempt_id = a.id
			AND aa.question_id = eq.question_id
		LEFT JOIN LATERAL (
			SELECT answer_key
			FROM question_versions
			WHERE question_id = q.id
			  AND is_active = TRUE
			  AND status = 'final'
			ORDER BY version_no DESC
			LIMIT 1
		) qv ON TRUE
		WHERE a.id = $1
		  AND eq.seq_no = $2
	`, attemptID, questionNo)

	var out AttemptQuestion
	var stimulus sql.NullString
	if err := r.Scan(
		&out.AttemptID,
		&out.QuestionID,
		&out.SeqNo,
		&out.QuestionType,
		&out.StemHTML,
		&stimulus,
		&out.AnswerKey,
		&out.AnswerPayload,
		&out.IsDoubt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrQuestionNotInExam
		}
		return nil, fmt.Errorf("query attempt question: %w", err)
	}
	if stimulus.Valid {
		out.StimulusHTML = &stimulus.String
	}

	optRows, err := s.db.QueryContext(ctx, `
		SELECT option_key, option_html
		FROM question_options
		WHERE question_id = $1
		ORDER BY option_key ASC
	`, out.QuestionID)
	if err != nil {
		return nil, fmt.Errorf("query question options: %w", err)
	}
	defer optRows.Close()

	for optRows.Next() {
		var opt QuestionOption
		if err := optRows.Scan(&opt.OptionKey, &opt.OptionHTML); err != nil {
			return nil, fmt.Errorf("scan question option: %w", err)
		}
		out.Options = append(out.Options, opt)
	}
	if err := optRows.Err(); err != nil {
		return nil, fmt.Errorf("iterate question options: %w", err)
	}

	return &out, nil
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

func (s *Service) LogAttemptEvent(ctx context.Context, input AttemptEventInput) (*AttemptEvent, error) {
	if input.AttemptID <= 0 {
		return nil, ErrAttemptNotFound
	}

	eventType := strings.TrimSpace(strings.ToLower(input.EventType))
	if eventType != "tab_blur" && eventType != "reconnect" && eventType != "rapid_refresh" && eventType != "fullscreen_exit" {
		return nil, ErrInvalidEventType
	}

	if _, err := s.loadAttemptRow(ctx, s.db, input.AttemptID); err != nil {
		return nil, err
	}

	if len(input.Payload) == 0 {
		input.Payload = json.RawMessage(`{}`)
	}

	row := s.db.QueryRowContext(ctx, `
		INSERT INTO attempt_events (attempt_id, event_type, payload, client_ts, server_ts, actor_user_id, created_at)
		VALUES ($1, $2, $3::jsonb, $4, now(), NULLIF($5, 0), now())
		RETURNING id, attempt_id, event_type, payload, client_ts, server_ts, actor_user_id, created_at
	`, input.AttemptID, eventType, []byte(input.Payload), input.ClientTS, input.ActorUserID)

	return scanAttemptEvent(row)
}

func (s *Service) ListAttemptEvents(ctx context.Context, attemptID int64, limit int) ([]AttemptEvent, error) {
	if attemptID <= 0 {
		return nil, ErrAttemptNotFound
	}
	if limit <= 0 || limit > 500 {
		limit = 200
	}

	if _, err := s.loadAttemptRow(ctx, s.db, attemptID); err != nil {
		return nil, err
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT id, attempt_id, event_type, payload, client_ts, server_ts, actor_user_id, created_at
		FROM attempt_events
		WHERE attempt_id = $1
		ORDER BY server_ts DESC, id DESC
		LIMIT $2
	`, attemptID, limit)
	if err != nil {
		return nil, fmt.Errorf("query attempt events: %w", err)
	}
	defer rows.Close()

	items := make([]AttemptEvent, 0)
	for rows.Next() {
		item, err := scanAttemptEvent(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, *item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate attempt events: %w", err)
	}
	return items, nil
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
		result := ScoreQuestion(ScoreInput{
			QuestionType:  ev.QuestionType,
			AnswerKey:     ev.AnswerKey,
			AnswerPayload: ev.Payload,
			CorrectKeys:   ev.CorrectKeys,
			Weight:        ev.Weight,
		})

		if result.Answered {
			answered++
		}
		if result.IsCorrect != nil && *result.IsCorrect {
			totalCorrect++
			score += result.EarnedScore
		} else if result.EarnedScore > 0 {
			score += result.EarnedScore
		}

		var isCorrectPtr interface{}
		if result.IsCorrect != nil {
			isCorrectPtr = *result.IsCorrect
		} else {
			isCorrectPtr = nil
		}

		feedback := map[string]interface{}{
			"selected":  result.Selected,
			"correct":   result.Correct,
			"reason":    result.Reason,
			"breakdown": result.Breakdown,
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
		`, row.ID, ev.QuestionID, result.EarnedScore, isCorrectPtr, feedbackJSON); err != nil {
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
			qn.question_type,
			eq.weight,
			COALESCE(aa.answer_payload, '{}'::jsonb) AS answer_payload,
			COALESCE(qv.answer_key, '{}'::jsonb) AS answer_key,
			COALESCE(
				json_agg(qo.option_key) FILTER (WHERE qo.is_correct),
				'[]'::json
			) AS correct_keys_json
		FROM exam_questions eq
		JOIN questions qn
			ON qn.id = eq.question_id
		LEFT JOIN attempt_answers aa
			ON aa.attempt_id = $1
			AND aa.question_id = eq.question_id
		LEFT JOIN LATERAL (
			SELECT answer_key
			FROM question_versions
			WHERE question_id = eq.question_id
			  AND is_active = TRUE
			  AND status = 'final'
			ORDER BY version_no DESC
			LIMIT 1
		) qv ON TRUE
		LEFT JOIN question_options qo
			ON qo.question_id = eq.question_id
		WHERE eq.exam_id = $2
		GROUP BY eq.question_id, qn.question_type, eq.weight, aa.answer_payload, qv.answer_key, eq.seq_no
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
		if err := rows.Scan(&r.QuestionID, &r.QuestionType, &r.Weight, &r.Payload, &r.AnswerKey, &correctKeysJSON); err != nil {
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
				if reason, ok := f["reason"].(string); ok {
					item.Reason = strings.TrimSpace(reason)
				}
				item.Breakdown = anyToStatementScores(f["breakdown"])
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

func scanAttemptEvent(scanner interface{ Scan(dest ...any) error }) (*AttemptEvent, error) {
	var out AttemptEvent
	var clientTS sql.NullTime
	var actorUserID sql.NullInt64
	if err := scanner.Scan(
		&out.ID,
		&out.AttemptID,
		&out.EventType,
		&out.Payload,
		&clientTS,
		&out.ServerTS,
		&actorUserID,
		&out.CreatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrAttemptNotFound
		}
		return nil, fmt.Errorf("scan attempt event: %w", err)
	}
	if clientTS.Valid {
		out.ClientTS = &clientTS.Time
	}
	if actorUserID.Valid {
		out.ActorUserID = &actorUserID.Int64
	}
	return &out, nil
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

func anyToStatementScores(v interface{}) []StatementScore {
	arr, ok := v.([]interface{})
	if !ok || len(arr) == 0 {
		return nil
	}

	out := make([]StatementScore, 0, len(arr))
	for _, it := range arr {
		obj, ok := it.(map[string]interface{})
		if !ok {
			continue
		}
		id, _ := obj["id"].(string)
		id = strings.TrimSpace(id)
		correct, ok := obj["correct"].(bool)
		if id == "" || !ok {
			continue
		}
		var answer *bool
		if val, ok := obj["answer"].(bool); ok {
			answer = &val
		}
		out = append(out, StatementScore{
			ID:      id,
			Correct: correct,
			Answer:  answer,
		})
	}
	if len(out) == 0 {
		return nil
	}
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
