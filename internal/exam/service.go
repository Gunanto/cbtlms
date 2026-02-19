package exam

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/hex"
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
	ErrSubjectNotFound    = errors.New("subject not found")
	ErrAttemptNotFound    = errors.New("attempt not found")
	ErrAttemptNotEditable = errors.New("attempt is not editable")
	ErrQuestionNotInExam  = errors.New("question not in exam")
	ErrAttemptForbidden   = errors.New("attempt forbidden")
	ErrAttemptNotFinal    = errors.New("attempt not final")
	ErrResultPolicyDenied = errors.New("result not available by review policy")
	ErrInvalidEventType   = errors.New("invalid event type")
	ErrExamTokenRequired  = errors.New("token ujian wajib diisi")
	ErrExamTokenInvalid   = errors.New("token ujian tidak valid")
	ErrExamTokenExpired   = errors.New("token ujian sudah kedaluwarsa")
	ErrExamNotAssigned    = errors.New("peserta tidak terdaftar pada ujian ini")
	ErrExamCodeExists     = errors.New("kode ujian sudah digunakan")
	ErrAssignmentFeature  = errors.New("fitur enroll ujian belum aktif di database")
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

type CreateSubjectInput struct {
	EducationLevel string
	SubjectType    string
	Name           string
}

type UpdateSubjectInput struct {
	ID             int64
	EducationLevel string
	SubjectType    string
	Name           string
}

type ExamOption struct {
	ID           int64      `json:"id"`
	Code         string     `json:"code"`
	Title        string     `json:"title"`
	SubjectID    int64      `json:"subject_id"`
	EndAt        *time.Time `json:"end_at,omitempty"`
	ReviewPolicy string     `json:"review_policy"`
}

type ExamAdminRecord struct {
	ID              int64      `json:"id"`
	Code            string     `json:"code"`
	Title           string     `json:"title"`
	SubjectID       int64      `json:"subject_id"`
	SubjectName     string     `json:"subject_name"`
	EducationLevel  string     `json:"education_level"`
	SubjectType     string     `json:"subject_type"`
	DurationMinutes int        `json:"duration_minutes"`
	StartAt         *time.Time `json:"start_at,omitempty"`
	EndAt           *time.Time `json:"end_at,omitempty"`
	ReviewPolicy    string     `json:"review_policy"`
	IsActive        bool       `json:"is_active"`
	CreatedBy       *int64     `json:"created_by,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	QuestionCount   int        `json:"question_count"`
	AssignedCount   int        `json:"assigned_count"`
}

type CreateExamInput struct {
	Code            string
	Title           string
	SubjectID       int64
	DurationMinutes int
	StartAt         *time.Time
	EndAt           *time.Time
	ReviewPolicy    string
	CreatedBy       int64
}

type UpdateExamInput struct {
	ID              int64
	Code            string
	Title           string
	SubjectID       int64
	DurationMinutes int
	StartAt         *time.Time
	EndAt           *time.Time
	ReviewPolicy    string
	IsActive        bool
}

type ExamTokenExam struct {
	ID             int64      `json:"id"`
	Code           string     `json:"code"`
	Title          string     `json:"title"`
	SubjectID      int64      `json:"subject_id"`
	SubjectName    string     `json:"subject_name"`
	EducationLevel string     `json:"education_level"`
	SubjectType    string     `json:"subject_type"`
	EndAt          *time.Time `json:"end_at,omitempty"`
}

type ExamAccessToken struct {
	ExamID      int64     `json:"exam_id"`
	Token       string    `json:"token"`
	TTLMinutes  int       `json:"ttl_minutes"`
	GeneratedAt time.Time `json:"generated_at"`
	ExpiresAt   time.Time `json:"expires_at"`
	GeneratedBy int64     `json:"generated_by"`
}

type ExamAssignmentUser struct {
	UserID      int64     `json:"user_id"`
	Username    string    `json:"username"`
	FullName    string    `json:"full_name"`
	Role        string    `json:"role"`
	SchoolName  *string   `json:"school_name,omitempty"`
	ClassName   *string   `json:"class_name,omitempty"`
	Status      string    `json:"status"`
	AssignedAt  time.Time `json:"assigned_at"`
	AssignedBy  *int64    `json:"assigned_by,omitempty"`
	AssignedByN *string   `json:"assigned_by_name,omitempty"`
}

type ReplaceExamAssignmentsInput struct {
	ExamID      int64
	UserIDs     []int64
	AssignedBy  int64
	AllowedRole string
}

type ReplaceExamAssignmentsByClassInput struct {
	ExamID     int64
	SchoolID   int64
	ClassID    int64
	AssignedBy int64
}

type ExamQuestionManageItem struct {
	ExamID        int64   `json:"exam_id"`
	QuestionID    int64   `json:"question_id"`
	SeqNo         int     `json:"seq_no"`
	Weight        float64 `json:"weight"`
	QuestionType  string  `json:"question_type"`
	StemPreview   string  `json:"stem_preview"`
	SubjectID     int64   `json:"subject_id"`
	SubjectName   string  `json:"subject_name"`
	VersionNo     *int    `json:"version_no,omitempty"`
	VersionStatus *string `json:"version_status,omitempty"`
}

type UpsertExamQuestionInput struct {
	ExamID     int64
	QuestionID int64
	SeqNo      int
	Weight     float64
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

func (s *Service) StartAttempt(ctx context.Context, examID, studentID int64, examToken string) (*Attempt, error) {
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
	var examTokenHash sql.NullString
	var examTokenExpires sql.NullTime
	if err := s.db.QueryRowContext(ctx, `
		SELECT duration_minutes, exam_token_hash, exam_token_expires_at
		FROM exams
		WHERE id = $1 AND is_active = TRUE
	`, examID).Scan(&duration, &examTokenHash, &examTokenExpires); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrExamNotFound
		}
		return nil, fmt.Errorf("query exam duration: %w", err)
	}
	var assignmentCount int
	if err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM exam_assignments
		WHERE exam_id = $1
		  AND status = 'active'
	`, examID).Scan(&assignmentCount); err != nil {
		if !isUndefinedTableErr(err, "exam_assignments") {
			return nil, fmt.Errorf("query assignment count: %w", err)
		}
		assignmentCount = 0
	}
	if assignmentCount > 0 {
		var assigned bool
		if err := s.db.QueryRowContext(ctx, `
			SELECT EXISTS(
				SELECT 1
				FROM exam_assignments
				WHERE exam_id = $1
				  AND user_id = $2
				  AND status = 'active'
			)
		`, examID, studentID).Scan(&assigned); err != nil {
			return nil, fmt.Errorf("check assignment: %w", err)
		}
		if !assigned {
			return nil, ErrExamNotAssigned
		}
	}
	if examTokenHash.Valid && strings.TrimSpace(examTokenHash.String) != "" {
		now := time.Now()
		if !examTokenExpires.Valid || now.After(examTokenExpires.Time) {
			return nil, ErrExamTokenExpired
		}
		inputToken := normalizeExamToken(examToken)
		if inputToken == "" {
			return nil, ErrExamTokenRequired
		}
		if subtle.ConstantTimeCompare([]byte(hashExamToken(inputToken)), []byte(examTokenHash.String)) != 1 {
			return nil, ErrExamTokenInvalid
		}
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

func (s *Service) ListExamsForToken(ctx context.Context) ([]ExamTokenExam, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT
			e.id,
			e.code,
			e.title,
			e.subject_id,
			s.name AS subject_name,
			s.education_level,
			s.subject_type,
			e.end_at
		FROM exams e
		JOIN subjects s ON s.id = e.subject_id
		WHERE e.is_active = TRUE
		ORDER BY e.created_at DESC, e.id DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("query exams for token: %w", err)
	}
	defer rows.Close()

	items := make([]ExamTokenExam, 0)
	for rows.Next() {
		var it ExamTokenExam
		var endAt sql.NullTime
		if err := rows.Scan(
			&it.ID,
			&it.Code,
			&it.Title,
			&it.SubjectID,
			&it.SubjectName,
			&it.EducationLevel,
			&it.SubjectType,
			&endAt,
		); err != nil {
			return nil, fmt.Errorf("scan exams for token: %w", err)
		}
		if endAt.Valid {
			it.EndAt = &endAt.Time
		}
		items = append(items, it)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate exams for token: %w", err)
	}
	return items, nil
}

func (s *Service) GenerateExamToken(ctx context.Context, examID, generatedBy int64, ttlMinutes int) (*ExamAccessToken, error) {
	if examID <= 0 {
		return nil, ErrInvalidInput
	}
	if generatedBy <= 0 {
		return nil, ErrInvalidInput
	}
	if ttlMinutes <= 0 {
		ttlMinutes = 120
	}
	if ttlMinutes > 1440 {
		ttlMinutes = 1440
	}

	token, err := randomExamToken(6)
	if err != nil {
		return nil, fmt.Errorf("generate exam token: %w", err)
	}
	tokenHash := hashExamToken(token)

	var generatedAt time.Time
	var expiresAt time.Time
	if err := s.db.QueryRowContext(ctx, `
		UPDATE exams
		SET exam_token_hash = $2,
			exam_token_expires_at = now() + make_interval(mins => $3),
			exam_token_generated_at = now(),
			exam_token_generated_by = $4
		WHERE id = $1
		  AND is_active = TRUE
		RETURNING exam_token_generated_at, exam_token_expires_at
	`, examID, tokenHash, ttlMinutes, generatedBy).Scan(&generatedAt, &expiresAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrExamNotFound
		}
		return nil, fmt.Errorf("update exam token: %w", err)
	}

	return &ExamAccessToken{
		ExamID:      examID,
		Token:       token,
		TTLMinutes:  ttlMinutes,
		GeneratedAt: generatedAt,
		ExpiresAt:   expiresAt,
		GeneratedBy: generatedBy,
	}, nil
}

func (s *Service) ListAdminExams(ctx context.Context, includeInactive bool) ([]ExamAdminRecord, error) {
	query := `
		SELECT
			e.id,
			e.code,
			e.title,
			e.subject_id,
			s.name AS subject_name,
			s.education_level,
			s.subject_type,
			e.duration_minutes,
			e.start_at,
			e.end_at,
			e.review_policy,
			e.is_active,
			e.created_by,
			e.created_at,
			COALESCE(eq.question_count, 0) AS question_count,
			COALESCE(ea.assigned_count, 0) AS assigned_count
		FROM exams e
		JOIN subjects s ON s.id = e.subject_id
		LEFT JOIN (
			SELECT exam_id, COUNT(*)::int AS question_count
			FROM exam_questions
			GROUP BY exam_id
		) eq ON eq.exam_id = e.id
			LEFT JOIN (
				SELECT exam_id, COUNT(*)::int AS assigned_count
				FROM exam_assignments
				WHERE status = 'active'
				GROUP BY exam_id
			) ea ON ea.exam_id = e.id
	`
	args := make([]any, 0, 1)
	if !includeInactive {
		query += " WHERE e.is_active = TRUE"
	}
	query += " ORDER BY e.created_at DESC, e.id DESC"

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		if isUndefinedTableErr(err, "exam_assignments") {
			query = `
				SELECT
					e.id,
					e.code,
					e.title,
					e.subject_id,
					s.name AS subject_name,
					s.education_level,
					s.subject_type,
					e.duration_minutes,
					e.start_at,
					e.end_at,
					e.review_policy,
					e.is_active,
					e.created_by,
					e.created_at,
					COALESCE(eq.question_count, 0) AS question_count,
					0::int AS assigned_count
				FROM exams e
				JOIN subjects s ON s.id = e.subject_id
				LEFT JOIN (
					SELECT exam_id, COUNT(*)::int AS question_count
					FROM exam_questions
					GROUP BY exam_id
				) eq ON eq.exam_id = e.id
			`
			if !includeInactive {
				query += " WHERE e.is_active = TRUE"
			}
			query += " ORDER BY e.created_at DESC, e.id DESC"
			rows, err = s.db.QueryContext(ctx, query)
		}
	}
	if err != nil {
		return nil, fmt.Errorf("list admin exams: %w", err)
	}
	defer rows.Close()

	items := make([]ExamAdminRecord, 0)
	for rows.Next() {
		var it ExamAdminRecord
		var startAt sql.NullTime
		var endAt sql.NullTime
		var createdBy sql.NullInt64
		if err := rows.Scan(
			&it.ID,
			&it.Code,
			&it.Title,
			&it.SubjectID,
			&it.SubjectName,
			&it.EducationLevel,
			&it.SubjectType,
			&it.DurationMinutes,
			&startAt,
			&endAt,
			&it.ReviewPolicy,
			&it.IsActive,
			&createdBy,
			&it.CreatedAt,
			&it.QuestionCount,
			&it.AssignedCount,
		); err != nil {
			return nil, fmt.Errorf("scan admin exam: %w", err)
		}
		if startAt.Valid {
			it.StartAt = &startAt.Time
		}
		if endAt.Valid {
			it.EndAt = &endAt.Time
		}
		if createdBy.Valid {
			it.CreatedBy = &createdBy.Int64
		}
		items = append(items, it)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate admin exams: %w", err)
	}
	return items, nil
}

func (s *Service) CreateExam(ctx context.Context, in CreateExamInput) (*ExamAdminRecord, error) {
	in.Code = strings.ToUpper(strings.TrimSpace(in.Code))
	in.Title = strings.TrimSpace(in.Title)
	in.ReviewPolicy = normalizeReviewPolicy(in.ReviewPolicy)
	if in.Title == "" || in.SubjectID <= 0 || in.CreatedBy <= 0 {
		return nil, ErrInvalidInput
	}
	if in.DurationMinutes <= 0 {
		in.DurationMinutes = s.defaultExamMinutes
	}
	insertOnce := func(code string) (int64, error) {
		var outID int64
		if err := s.db.QueryRowContext(ctx, `
			INSERT INTO exams (
				code, title, subject_id, duration_minutes,
				start_at, end_at, review_policy, is_active, created_by, created_at
			) VALUES ($1,$2,$3,$4,$5,$6,$7,TRUE,$8,now())
			RETURNING id
		`, code, in.Title, in.SubjectID, in.DurationMinutes, in.StartAt, in.EndAt, in.ReviewPolicy, in.CreatedBy).Scan(&outID); err != nil {
			return 0, err
		}
		return outID, nil
	}

	if in.Code != "" {
		outID, err := insertOnce(in.Code)
		if err != nil {
			if strings.Contains(strings.ToLower(err.Error()), "exams_code_key") {
				return nil, ErrExamCodeExists
			}
			return nil, fmt.Errorf("create exam: %w", err)
		}
		return s.GetExamAdminByID(ctx, outID)
	}

	for i := 0; i < 8; i++ {
		token, err := randomExamToken(4)
		if err != nil {
			return nil, fmt.Errorf("generate exam code: %w", err)
		}
		autoCode := "EXAM-" + time.Now().Format("20060102") + "-" + token
		outID, err := insertOnce(autoCode)
		if err != nil {
			if strings.Contains(strings.ToLower(err.Error()), "exams_code_key") {
				continue
			}
			return nil, fmt.Errorf("create exam: %w", err)
		}
		return s.GetExamAdminByID(ctx, outID)
	}
	return nil, fmt.Errorf("create exam: gagal generate kode ujian unik")
}

func (s *Service) UpdateExam(ctx context.Context, in UpdateExamInput) (*ExamAdminRecord, error) {
	in.Code = strings.ToUpper(strings.TrimSpace(in.Code))
	in.Title = strings.TrimSpace(in.Title)
	in.ReviewPolicy = normalizeReviewPolicy(in.ReviewPolicy)
	if in.ID <= 0 || in.Code == "" || in.Title == "" || in.SubjectID <= 0 {
		return nil, ErrInvalidInput
	}
	if in.DurationMinutes <= 0 {
		in.DurationMinutes = s.defaultExamMinutes
	}
	var outID int64
	if err := s.db.QueryRowContext(ctx, `
		UPDATE exams
		SET code = $2,
			title = $3,
			subject_id = $4,
			duration_minutes = $5,
			start_at = $6,
			end_at = $7,
			review_policy = $8,
			is_active = $9
		WHERE id = $1
		RETURNING id
	`, in.ID, in.Code, in.Title, in.SubjectID, in.DurationMinutes, in.StartAt, in.EndAt, in.ReviewPolicy, in.IsActive).Scan(&outID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrExamNotFound
		}
		if strings.Contains(strings.ToLower(err.Error()), "exams_code_key") {
			return nil, ErrExamCodeExists
		}
		return nil, fmt.Errorf("update exam: %w", err)
	}
	return s.GetExamAdminByID(ctx, outID)
}

func (s *Service) DeleteExam(ctx context.Context, examID int64) error {
	if examID <= 0 {
		return ErrInvalidInput
	}
	var outID int64
	if err := s.db.QueryRowContext(ctx, `
		UPDATE exams
		SET is_active = FALSE
		WHERE id = $1
		RETURNING id
	`, examID).Scan(&outID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrExamNotFound
		}
		return fmt.Errorf("delete exam: %w", err)
	}
	return nil
}

func (s *Service) GetExamAdminByID(ctx context.Context, examID int64) (*ExamAdminRecord, error) {
	if examID <= 0 {
		return nil, ErrInvalidInput
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT
			e.id,
			e.code,
			e.title,
			e.subject_id,
			s.name AS subject_name,
			s.education_level,
			s.subject_type,
			e.duration_minutes,
			e.start_at,
			e.end_at,
			e.review_policy,
			e.is_active,
			e.created_by,
			e.created_at,
			COALESCE(eq.question_count, 0) AS question_count,
			COALESCE(ea.assigned_count, 0) AS assigned_count
		FROM exams e
		JOIN subjects s ON s.id = e.subject_id
		LEFT JOIN (
			SELECT exam_id, COUNT(*)::int AS question_count
			FROM exam_questions
			GROUP BY exam_id
		) eq ON eq.exam_id = e.id
			LEFT JOIN (
				SELECT exam_id, COUNT(*)::int AS assigned_count
				FROM exam_assignments
				WHERE status = 'active'
				GROUP BY exam_id
			) ea ON ea.exam_id = e.id
			WHERE e.id = $1
	`, examID)
	if err != nil {
		if isUndefinedTableErr(err, "exam_assignments") {
			rows, err = s.db.QueryContext(ctx, `
				SELECT
					e.id,
					e.code,
					e.title,
					e.subject_id,
					s.name AS subject_name,
					s.education_level,
					s.subject_type,
					e.duration_minutes,
					e.start_at,
					e.end_at,
					e.review_policy,
					e.is_active,
					e.created_by,
					e.created_at,
					COALESCE(eq.question_count, 0) AS question_count,
					0::int AS assigned_count
				FROM exams e
				JOIN subjects s ON s.id = e.subject_id
				LEFT JOIN (
					SELECT exam_id, COUNT(*)::int AS question_count
					FROM exam_questions
					GROUP BY exam_id
				) eq ON eq.exam_id = e.id
				WHERE e.id = $1
			`, examID)
		}
	}
	if err != nil {
		return nil, fmt.Errorf("get exam admin: %w", err)
	}
	defer rows.Close()
	if !rows.Next() {
		return nil, ErrExamNotFound
	}
	var it ExamAdminRecord
	var startAt sql.NullTime
	var endAt sql.NullTime
	var createdBy sql.NullInt64
	if err := rows.Scan(
		&it.ID,
		&it.Code,
		&it.Title,
		&it.SubjectID,
		&it.SubjectName,
		&it.EducationLevel,
		&it.SubjectType,
		&it.DurationMinutes,
		&startAt,
		&endAt,
		&it.ReviewPolicy,
		&it.IsActive,
		&createdBy,
		&it.CreatedAt,
		&it.QuestionCount,
		&it.AssignedCount,
	); err != nil {
		return nil, fmt.Errorf("scan exam admin: %w", err)
	}
	if startAt.Valid {
		it.StartAt = &startAt.Time
	}
	if endAt.Valid {
		it.EndAt = &endAt.Time
	}
	if createdBy.Valid {
		it.CreatedBy = &createdBy.Int64
	}
	return &it, nil
}

func (s *Service) ListExamAssignments(ctx context.Context, examID int64) ([]ExamAssignmentUser, error) {
	if examID <= 0 {
		return nil, ErrInvalidInput
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT
			u.id,
			u.username,
			u.full_name,
			u.role,
			sch.name AS school_name,
			cls.name AS class_name,
			ea.status,
			ea.assigned_at,
			ea.assigned_by,
			ab.full_name AS assigned_by_name
		FROM exam_assignments ea
		JOIN users u ON u.id = ea.user_id
		LEFT JOIN enrollments en ON en.user_id = u.id AND en.status = 'active'
		LEFT JOIN schools sch ON sch.id = en.school_id
		LEFT JOIN classes cls ON cls.id = en.class_id
		LEFT JOIN users ab ON ab.id = ea.assigned_by
		WHERE ea.exam_id = $1
		  AND ea.status = 'active'
		ORDER BY u.role, u.full_name, u.username
	`, examID)
	if err != nil {
		if isUndefinedTableErr(err, "exam_assignments") {
			return nil, ErrAssignmentFeature
		}
		return nil, fmt.Errorf("list exam assignments: %w", err)
	}
	defer rows.Close()
	items := make([]ExamAssignmentUser, 0)
	for rows.Next() {
		var it ExamAssignmentUser
		var schoolName sql.NullString
		var className sql.NullString
		var assignedBy sql.NullInt64
		var assignedByName sql.NullString
		if err := rows.Scan(
			&it.UserID,
			&it.Username,
			&it.FullName,
			&it.Role,
			&schoolName,
			&className,
			&it.Status,
			&it.AssignedAt,
			&assignedBy,
			&assignedByName,
		); err != nil {
			return nil, fmt.Errorf("scan exam assignment: %w", err)
		}
		if schoolName.Valid {
			it.SchoolName = &schoolName.String
		}
		if className.Valid {
			it.ClassName = &className.String
		}
		if assignedBy.Valid {
			it.AssignedBy = &assignedBy.Int64
		}
		if assignedByName.Valid {
			it.AssignedByN = &assignedByName.String
		}
		items = append(items, it)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate exam assignments: %w", err)
	}
	return items, nil
}

func (s *Service) ReplaceExamAssignments(ctx context.Context, in ReplaceExamAssignmentsInput) ([]ExamAssignmentUser, error) {
	if in.ExamID <= 0 || in.AssignedBy <= 0 {
		return nil, ErrInvalidInput
	}
	if _, err := s.GetExamAdminByID(ctx, in.ExamID); err != nil {
		return nil, err
	}

	ids := make([]int64, 0, len(in.UserIDs))
	seen := map[int64]struct{}{}
	for _, id := range in.UserIDs {
		if id <= 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin assignment tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, `
		UPDATE exam_assignments
		SET status = 'inactive',
			updated_at = now()
		WHERE exam_id = $1
	`, in.ExamID); err != nil {
		if isUndefinedTableErr(err, "exam_assignments") {
			return nil, ErrAssignmentFeature
		}
		return nil, fmt.Errorf("deactivate old exam assignments: %w", err)
	}

	for _, userID := range ids {
		var role string
		if err := tx.QueryRowContext(ctx, `
			SELECT role
			FROM users
			WHERE id = $1
			  AND is_active = TRUE
		`, userID).Scan(&role); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				continue
			}
			return nil, fmt.Errorf("load assignment user role: %w", err)
		}
		if role != "guru" && role != "siswa" {
			continue
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO exam_assignments (
				exam_id, user_id, status, assigned_by, assigned_at, created_at, updated_at
			) VALUES ($1, $2, 'active', $3, now(), now(), now())
			ON CONFLICT (exam_id, user_id)
			DO UPDATE SET
				status = 'active',
				assigned_by = EXCLUDED.assigned_by,
				assigned_at = now(),
				updated_at = now()
		`, in.ExamID, userID, in.AssignedBy); err != nil {
			return nil, fmt.Errorf("upsert exam assignment: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit assignment tx: %w", err)
	}
	return s.ListExamAssignments(ctx, in.ExamID)
}

func (s *Service) ReplaceExamAssignmentsByClass(ctx context.Context, in ReplaceExamAssignmentsByClassInput) ([]ExamAssignmentUser, error) {
	if in.ExamID <= 0 || in.SchoolID <= 0 || in.ClassID <= 0 || in.AssignedBy <= 0 {
		return nil, ErrInvalidInput
	}
	if _, err := s.GetExamAdminByID(ctx, in.ExamID); err != nil {
		return nil, err
	}

	var classValid bool
	if err := s.db.QueryRowContext(ctx, `
		SELECT EXISTS(
			SELECT 1
			FROM classes c
			JOIN schools s ON s.id = c.school_id
			WHERE c.id = $1
			  AND c.school_id = $2
			  AND c.is_active = TRUE
			  AND s.is_active = TRUE
		)
	`, in.ClassID, in.SchoolID).Scan(&classValid); err != nil {
		return nil, fmt.Errorf("validate class for exam assignment: %w", err)
	}
	if !classValid {
		return nil, ErrInvalidInput
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT DISTINCT e.user_id
		FROM enrollments e
		JOIN users u ON u.id = e.user_id
		WHERE e.school_id = $1
		  AND e.class_id = $2
		  AND e.status = 'active'
		  AND u.is_active = TRUE
		  AND u.role = 'siswa'
		ORDER BY e.user_id
	`, in.SchoolID, in.ClassID)
	if err != nil {
		return nil, fmt.Errorf("list class users for exam assignment: %w", err)
	}
	defer rows.Close()

	userIDs := make([]int64, 0)
	for rows.Next() {
		var uid int64
		if err := rows.Scan(&uid); err != nil {
			return nil, fmt.Errorf("scan class user for exam assignment: %w", err)
		}
		userIDs = append(userIDs, uid)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate class users for exam assignment: %w", err)
	}

	return s.ReplaceExamAssignments(ctx, ReplaceExamAssignmentsInput{
		ExamID:     in.ExamID,
		UserIDs:    userIDs,
		AssignedBy: in.AssignedBy,
	})
}

func (s *Service) ListExamQuestions(ctx context.Context, examID int64) ([]ExamQuestionManageItem, error) {
	if examID <= 0 {
		return nil, ErrInvalidInput
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT
			eq.exam_id,
			eq.question_id,
			eq.seq_no,
			eq.weight,
			q.question_type,
			LEFT(COALESCE(qv.stem_html, q.stem_html, ''), 200) AS stem_preview,
			q.subject_id,
			s.name AS subject_name,
			qv.version_no,
			qv.status
		FROM exam_questions eq
		JOIN questions q ON q.id = eq.question_id
		JOIN subjects s ON s.id = q.subject_id
		LEFT JOIN LATERAL (
			SELECT version_no, stem_html, status
			FROM question_versions
			WHERE question_id = q.id
			  AND is_active = TRUE
			ORDER BY version_no DESC
			LIMIT 1
		) qv ON TRUE
		WHERE eq.exam_id = $1
		ORDER BY eq.seq_no ASC
	`, examID)
	if err != nil {
		return nil, fmt.Errorf("list exam questions: %w", err)
	}
	defer rows.Close()
	items := make([]ExamQuestionManageItem, 0)
	for rows.Next() {
		var it ExamQuestionManageItem
		var versionNo sql.NullInt64
		var versionStatus sql.NullString
		if err := rows.Scan(
			&it.ExamID,
			&it.QuestionID,
			&it.SeqNo,
			&it.Weight,
			&it.QuestionType,
			&it.StemPreview,
			&it.SubjectID,
			&it.SubjectName,
			&versionNo,
			&versionStatus,
		); err != nil {
			return nil, fmt.Errorf("scan exam question: %w", err)
		}
		if versionNo.Valid {
			v := int(versionNo.Int64)
			it.VersionNo = &v
		}
		if versionStatus.Valid {
			it.VersionStatus = &versionStatus.String
		}
		items = append(items, it)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate exam questions: %w", err)
	}
	return items, nil
}

func (s *Service) UpsertExamQuestion(ctx context.Context, in UpsertExamQuestionInput) (*ExamQuestionManageItem, error) {
	if in.ExamID <= 0 || in.QuestionID <= 0 {
		return nil, ErrInvalidInput
	}
	if in.Weight <= 0 {
		in.Weight = 1
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin upsert exam question tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var examExists bool
	if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM exams WHERE id = $1)`, in.ExamID).Scan(&examExists); err != nil {
		return nil, fmt.Errorf("check exam exists: %w", err)
	}
	if !examExists {
		return nil, ErrExamNotFound
	}
	var questionExists bool
	if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM questions WHERE id = $1)`, in.QuestionID).Scan(&questionExists); err != nil {
		return nil, fmt.Errorf("check question exists: %w", err)
	}
	if !questionExists {
		return nil, ErrQuestionNotInExam
	}

	seqNo := in.SeqNo
	if seqNo <= 0 {
		if err := tx.QueryRowContext(ctx, `
			SELECT COALESCE(MAX(seq_no), 0) + 1
			FROM exam_questions
			WHERE exam_id = $1
		`, in.ExamID).Scan(&seqNo); err != nil {
			return nil, fmt.Errorf("next seq_no exam question: %w", err)
		}
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO exam_questions (exam_id, question_id, seq_no, weight)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (exam_id, question_id)
		DO UPDATE SET
			seq_no = EXCLUDED.seq_no,
			weight = EXCLUDED.weight
	`, in.ExamID, in.QuestionID, seqNo, in.Weight); err != nil {
		return nil, fmt.Errorf("upsert exam question: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit upsert exam question tx: %w", err)
	}

	items, err := s.ListExamQuestions(ctx, in.ExamID)
	if err != nil {
		return nil, err
	}
	for _, it := range items {
		if it.QuestionID == in.QuestionID {
			cloned := it
			return &cloned, nil
		}
	}
	return nil, ErrQuestionNotInExam
}

func (s *Service) DeleteExamQuestion(ctx context.Context, examID, questionID int64) error {
	if examID <= 0 || questionID <= 0 {
		return ErrInvalidInput
	}
	res, err := s.db.ExecContext(ctx, `
		DELETE FROM exam_questions
		WHERE exam_id = $1
		  AND question_id = $2
	`, examID, questionID)
	if err != nil {
		return fmt.Errorf("delete exam question: %w", err)
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		return ErrQuestionNotInExam
	}
	return nil
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

func (s *Service) CreateSubject(ctx context.Context, in CreateSubjectInput) (*SubjectOption, error) {
	level := strings.TrimSpace(in.EducationLevel)
	subjectType := strings.TrimSpace(in.SubjectType)
	name := strings.TrimSpace(in.Name)
	if level == "" || subjectType == "" || name == "" {
		return nil, ErrInvalidInput
	}
	var out SubjectOption
	if err := s.db.QueryRowContext(ctx, `
		INSERT INTO subjects (education_level, subject_type, name, is_active)
		VALUES ($1, $2, $3, TRUE)
		RETURNING id, education_level, subject_type, name
	`, level, subjectType, name).Scan(&out.ID, &out.EducationLevel, &out.SubjectType, &out.Name); err != nil {
		return nil, fmt.Errorf("insert subject: %w", err)
	}
	return &out, nil
}

func (s *Service) UpdateSubject(ctx context.Context, in UpdateSubjectInput) (*SubjectOption, error) {
	if in.ID <= 0 {
		return nil, ErrInvalidInput
	}
	level := strings.TrimSpace(in.EducationLevel)
	subjectType := strings.TrimSpace(in.SubjectType)
	name := strings.TrimSpace(in.Name)
	if level == "" || subjectType == "" || name == "" {
		return nil, ErrInvalidInput
	}
	var out SubjectOption
	if err := s.db.QueryRowContext(ctx, `
		UPDATE subjects
		SET education_level = $2,
			subject_type = $3,
			name = $4
		WHERE id = $1
		  AND is_active = TRUE
		RETURNING id, education_level, subject_type, name
	`, in.ID, level, subjectType, name).Scan(&out.ID, &out.EducationLevel, &out.SubjectType, &out.Name); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrSubjectNotFound
		}
		return nil, fmt.Errorf("update subject: %w", err)
	}
	return &out, nil
}

func normalizeExamToken(token string) string {
	return strings.ToUpper(strings.TrimSpace(token))
}

func normalizeReviewPolicy(input string) string {
	v := strings.TrimSpace(strings.ToLower(input))
	switch v {
	case "after_submit", "immediate", "disabled", "after_exam_end":
		return v
	case "after_end":
		return "after_exam_end"
	default:
		return "after_submit"
	}
}

func hashExamToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func randomExamToken(length int) (string, error) {
	if length <= 0 {
		length = 6
	}
	const alphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	buf := make([]byte, length)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	out := make([]byte, length)
	for i := range buf {
		out[i] = alphabet[int(buf[i])%len(alphabet)]
	}
	return string(out), nil
}

func isUndefinedTableErr(err error, table string) bool {
	if err == nil {
		return false
	}
	lower := strings.ToLower(err.Error())
	if table == "" {
		return strings.Contains(lower, "does not exist") || strings.Contains(lower, "undefined_table")
	}
	return strings.Contains(lower, "relation \""+strings.ToLower(table)+"\" does not exist") ||
		(strings.Contains(lower, "undefined_table") && strings.Contains(lower, strings.ToLower(table)))
}

func (s *Service) DeleteSubject(ctx context.Context, subjectID int64) error {
	if subjectID <= 0 {
		return ErrInvalidInput
	}
	var id int64
	if err := s.db.QueryRowContext(ctx, `
		UPDATE subjects
		SET is_active = FALSE
		WHERE id = $1
		  AND is_active = TRUE
		RETURNING id
	`, subjectID).Scan(&id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrSubjectNotFound
		}
		return fmt.Errorf("delete subject: %w", err)
	}
	return nil
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
