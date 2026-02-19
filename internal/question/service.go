package question

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	ErrInvalidInput       = errors.New("invalid input")
	ErrSubjectNotFound    = errors.New("subject not found")
	ErrStimulusNotFound   = errors.New("stimulus not found")
	ErrExamNotFound       = errors.New("exam not found")
	ErrQuestionNotFound   = errors.New("question not found")
	ErrParallelNotFound   = errors.New("question parallel not found")
	ErrQuestionNotInExam  = errors.New("question not in exam")
	ErrVersionNotFound    = errors.New("question version not found")
	ErrReviewTaskNotFound = errors.New("review task not found")
	ErrReviewForbidden    = errors.New("review forbidden")
)

type Service struct {
	db *sql.DB
}

type CreateStimulusInput struct {
	SubjectID    int64
	Title        string
	StimulusType string
	Content      json.RawMessage
	CreatedBy    int64
}

type Stimulus struct {
	ID           int64           `json:"id"`
	SubjectID    int64           `json:"subject_id"`
	Title        string          `json:"title"`
	StimulusType string          `json:"stimulus_type"`
	Content      json.RawMessage `json:"content"`
	IsActive     bool            `json:"is_active"`
	CreatedBy    *int64          `json:"created_by,omitempty"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
}

type UpdateStimulusInput struct {
	ID           int64
	SubjectID    int64
	Title        string
	StimulusType string
	Content      json.RawMessage
}

type CreateQuestionInput struct {
	SubjectID      int64
	QuestionType   string
	Title          string
	Indicator      string
	Material       string
	Objective      string
	CognitiveLevel string
	Difficulty     *int
	CreatedBy      int64
}

type QuestionBlueprint struct {
	ID             int64           `json:"id"`
	SubjectID      int64           `json:"subject_id"`
	EducationLevel string          `json:"education_level"`
	SubjectType    string          `json:"subject_type"`
	SubjectName    string          `json:"subject_name"`
	QuestionType   string          `json:"question_type"`
	Title          string          `json:"title"`
	Indicator      string          `json:"indicator"`
	Material       string          `json:"material"`
	Objective      string          `json:"objective"`
	CognitiveLevel string          `json:"cognitive_level"`
	Difficulty     *int            `json:"difficulty,omitempty"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
	Metadata       json.RawMessage `json:"metadata,omitempty"`
}

type CreateQuestionVersionInput struct {
	QuestionID      int64
	StimulusID      *int64
	StemHTML        *string
	ExplanationHTML *string
	HintHTML        *string
	AnswerKey       json.RawMessage
	Options         []QuestionOptionInput
	DurationSeconds *int
	Weight          *float64
	ChangeNote      *string
	CreatedBy       int64
}

type QuestionOptionInput struct {
	OptionKey  string `json:"option_key"`
	OptionHTML string `json:"option_html"`
}

type QuestionVersion struct {
	ID              int64           `json:"id"`
	QuestionID      int64           `json:"question_id"`
	VersionNo       int             `json:"version_no"`
	StimulusID      *int64          `json:"stimulus_id,omitempty"`
	StemHTML        string          `json:"stem_html"`
	ExplanationHTML *string         `json:"explanation_html,omitempty"`
	HintHTML        *string         `json:"hint_html,omitempty"`
	AnswerKey       json.RawMessage `json:"answer_key"`
	Status          string          `json:"status"`
	IsPublic        bool            `json:"is_public"`
	IsActive        bool            `json:"is_active"`
	DurationSeconds *int            `json:"duration_seconds,omitempty"`
	Weight          float64         `json:"weight"`
	ChangeNote      *string         `json:"change_note,omitempty"`
	CreatedBy       *int64          `json:"created_by,omitempty"`
	CreatedAt       time.Time       `json:"created_at"`
}

type CreateQuestionParallelInput struct {
	ExamID        int64
	QuestionID    int64
	ParallelGroup string
	ParallelOrder int
	ParallelLabel string
}

type UpdateQuestionParallelInput struct {
	ExamID        int64
	ParallelID    int64
	QuestionID    int64
	ParallelGroup string
	ParallelOrder int
	ParallelLabel string
}

type QuestionParallel struct {
	ID            int64     `json:"id"`
	ExamID        int64     `json:"exam_id"`
	QuestionID    int64     `json:"question_id"`
	ParallelGroup string    `json:"parallel_group"`
	ParallelOrder int       `json:"parallel_order"`
	ParallelLabel string    `json:"parallel_label"`
	IsActive      bool      `json:"is_active"`
	CreatedAt     time.Time `json:"created_at"`
}

type CreateReviewTaskInput struct {
	QuestionVersionID int64
	ExamID            *int64
	ReviewerID        int64
	Note              string
}

type ReviewDecisionInput struct {
	TaskID       int64
	ReviewerID   int64
	IsPrivileged bool
	Status       string
	Note         string
}

type ReviewTask struct {
	ID                int64      `json:"id"`
	QuestionVersionID int64      `json:"question_version_id"`
	QuestionID        int64      `json:"question_id"`
	ExamID            *int64     `json:"exam_id,omitempty"`
	ReviewerID        int64      `json:"reviewer_id"`
	Status            string     `json:"status"`
	AssignedAt        time.Time  `json:"assigned_at"`
	ReviewedAt        *time.Time `json:"reviewed_at,omitempty"`
	Note              *string    `json:"note,omitempty"`
}

type ReviewComment struct {
	ID           int64     `json:"id"`
	ReviewTaskID int64     `json:"review_task_id"`
	AuthorID     int64     `json:"author_id"`
	CommentType  string    `json:"comment_type"`
	CommentText  string    `json:"comment_text"`
	CreatedAt    time.Time `json:"created_at"`
}

type QuestionReview struct {
	Task     ReviewTask      `json:"task"`
	Comments []ReviewComment `json:"comments"`
}

func NewService(db *sql.DB) *Service {
	return &Service{db: db}
}

func normalizeQuestionType(v string) string {
	v = strings.TrimSpace(strings.ToLower(v))
	switch v {
	case "pg_tunggal", "multi_jawaban", "benar_salah_pernyataan":
		return v
	default:
		return ""
	}
}

func nullableDifficulty(v *int) any {
	if v == nil {
		return nil
	}
	return *v
}

func populateBlueprintMetadata(out *QuestionBlueprint, raw json.RawMessage) {
	if out == nil || len(raw) == 0 {
		return
	}
	var obj map[string]any
	if err := json.Unmarshal(raw, &obj); err != nil {
		return
	}
	getText := func(key string) string {
		rawValue, ok := obj[key]
		if !ok || rawValue == nil {
			return ""
		}
		s, ok := rawValue.(string)
		if !ok {
			return ""
		}
		return strings.TrimSpace(s)
	}
	out.Title = getText("title")
	out.Indicator = getText("indicator")
	out.Material = getText("material")
	out.Objective = getText("objective")
	out.CognitiveLevel = getText("cognitive_level")
}

func (s *Service) CreateQuestion(ctx context.Context, in CreateQuestionInput) (*QuestionBlueprint, error) {
	in.QuestionType = normalizeQuestionType(in.QuestionType)
	in.Title = strings.TrimSpace(in.Title)
	in.Indicator = strings.TrimSpace(in.Indicator)
	in.Material = strings.TrimSpace(in.Material)
	in.Objective = strings.TrimSpace(in.Objective)
	in.CognitiveLevel = strings.TrimSpace(in.CognitiveLevel)

	if in.SubjectID <= 0 || in.QuestionType == "" || in.Title == "" {
		return nil, ErrInvalidInput
	}
	if in.Difficulty != nil && (*in.Difficulty < 1 || *in.Difficulty > 5) {
		return nil, fmt.Errorf("%w: difficulty must be between 1 and 5", ErrInvalidInput)
	}

	var subject struct {
		educationLevel string
		subjectType    string
		name           string
	}
	if err := s.db.QueryRowContext(ctx, `
		SELECT education_level, subject_type, name
		FROM subjects
		WHERE id = $1 AND is_active = TRUE
	`, in.SubjectID).Scan(&subject.educationLevel, &subject.subjectType, &subject.name); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrSubjectNotFound
		}
		return nil, fmt.Errorf("load subject: %w", err)
	}

	metaRaw, err := json.Marshal(map[string]any{
		"title":           in.Title,
		"indicator":       in.Indicator,
		"material":        in.Material,
		"objective":       in.Objective,
		"cognitive_level": in.CognitiveLevel,
		"created_via":     "guru_dashboard",
	})
	if err != nil {
		return nil, fmt.Errorf("marshal metadata: %w", err)
	}

	stem := "<p>Draft naskah soal: " + in.Title + "</p>"
	row := s.db.QueryRowContext(ctx, `
		INSERT INTO questions (
			subject_id, question_type, stem_html, stimulus_html, difficulty,
			metadata, is_active, version, created_at, updated_at
		) VALUES (
			$1, $2, $3, NULL, $4, $5::jsonb, TRUE, 1, now(), now()
		)
		RETURNING id, subject_id, question_type, difficulty, metadata, created_at, updated_at
	`, in.SubjectID, in.QuestionType, stem, nullableDifficulty(in.Difficulty), []byte(metaRaw))

	var out QuestionBlueprint
	var difficulty sql.NullInt64
	if err := row.Scan(
		&out.ID,
		&out.SubjectID,
		&out.QuestionType,
		&difficulty,
		&out.Metadata,
		&out.CreatedAt,
		&out.UpdatedAt,
	); err != nil {
		return nil, fmt.Errorf("insert question: %w", err)
	}
	if difficulty.Valid {
		d := int(difficulty.Int64)
		out.Difficulty = &d
	}
	out.EducationLevel = subject.educationLevel
	out.SubjectType = subject.subjectType
	out.SubjectName = subject.name
	populateBlueprintMetadata(&out, out.Metadata)
	return &out, nil
}

func (s *Service) ListQuestions(ctx context.Context, subjectID int64) ([]QuestionBlueprint, error) {
	query := `
		SELECT q.id, q.subject_id, s.education_level, s.subject_type, s.name,
			q.question_type, q.difficulty, q.metadata, q.created_at, q.updated_at
		FROM questions q
		JOIN subjects s ON s.id = q.subject_id
		WHERE q.is_active = TRUE
	`
	args := make([]any, 0, 1)
	if subjectID > 0 {
		query += ` AND q.subject_id = $1`
		args = append(args, subjectID)
	}
	query += ` ORDER BY q.id DESC`

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query questions: %w", err)
	}
	defer rows.Close()

	items := make([]QuestionBlueprint, 0)
	for rows.Next() {
		var out QuestionBlueprint
		var difficulty sql.NullInt64
		if err := rows.Scan(
			&out.ID,
			&out.SubjectID,
			&out.EducationLevel,
			&out.SubjectType,
			&out.SubjectName,
			&out.QuestionType,
			&difficulty,
			&out.Metadata,
			&out.CreatedAt,
			&out.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan question: %w", err)
		}
		if difficulty.Valid {
			d := int(difficulty.Int64)
			out.Difficulty = &d
		}
		populateBlueprintMetadata(&out, out.Metadata)
		items = append(items, out)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate questions: %w", err)
	}
	return items, nil
}

func (s *Service) CreateStimulus(ctx context.Context, in CreateStimulusInput) (*Stimulus, error) {
	in.Title = strings.TrimSpace(in.Title)
	in.StimulusType = strings.ToLower(strings.TrimSpace(in.StimulusType))

	if in.SubjectID <= 0 || in.Title == "" || in.StimulusType == "" {
		return nil, ErrInvalidInput
	}
	if in.StimulusType != "single" && in.StimulusType != "multiteks" {
		return nil, fmt.Errorf("%w: stimulus_type must be single or multiteks", ErrInvalidInput)
	}

	if len(in.Content) == 0 {
		in.Content = json.RawMessage(`{}`)
	}
	if err := validateStimulusContent(in.StimulusType, in.Content); err != nil {
		return nil, err
	}

	var subjectExists bool
	if err := s.db.QueryRowContext(ctx, `
		SELECT EXISTS(SELECT 1 FROM subjects WHERE id = $1 AND is_active = TRUE)
	`, in.SubjectID).Scan(&subjectExists); err != nil {
		return nil, fmt.Errorf("check subject: %w", err)
	}
	if !subjectExists {
		return nil, ErrSubjectNotFound
	}

	row := s.db.QueryRowContext(ctx, `
		INSERT INTO stimuli (subject_id, title, stimulus_type, content, created_by, created_at, updated_at)
		VALUES ($1, $2, $3, $4::jsonb, NULLIF($5, 0), now(), now())
		RETURNING id, subject_id, title, stimulus_type, content, is_active, created_by, created_at, updated_at
	`, in.SubjectID, in.Title, in.StimulusType, []byte(in.Content), in.CreatedBy)

	var out Stimulus
	var createdBy sql.NullInt64
	if err := row.Scan(
		&out.ID,
		&out.SubjectID,
		&out.Title,
		&out.StimulusType,
		&out.Content,
		&out.IsActive,
		&createdBy,
		&out.CreatedAt,
		&out.UpdatedAt,
	); err != nil {
		return nil, fmt.Errorf("insert stimulus: %w", err)
	}
	if createdBy.Valid {
		out.CreatedBy = &createdBy.Int64
	}
	return &out, nil
}

func (s *Service) ListStimuliBySubject(ctx context.Context, subjectID int64) ([]Stimulus, error) {
	if subjectID <= 0 {
		return nil, ErrInvalidInput
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT id, subject_id, title, stimulus_type, content, is_active, created_by, created_at, updated_at
		FROM stimuli
		WHERE subject_id = $1 AND is_active = TRUE
		ORDER BY id DESC
	`, subjectID)
	if err != nil {
		return nil, fmt.Errorf("query stimuli: %w", err)
	}
	defer rows.Close()

	items := make([]Stimulus, 0)
	for rows.Next() {
		var item Stimulus
		var createdBy sql.NullInt64
		if err := rows.Scan(
			&item.ID,
			&item.SubjectID,
			&item.Title,
			&item.StimulusType,
			&item.Content,
			&item.IsActive,
			&createdBy,
			&item.CreatedAt,
			&item.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan stimulus: %w", err)
		}
		if createdBy.Valid {
			item.CreatedBy = &createdBy.Int64
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate stimuli: %w", err)
	}
	return items, nil
}

func (s *Service) UpdateStimulus(ctx context.Context, in UpdateStimulusInput) (*Stimulus, error) {
	in.Title = strings.TrimSpace(in.Title)
	in.StimulusType = strings.ToLower(strings.TrimSpace(in.StimulusType))
	if in.ID <= 0 || in.SubjectID <= 0 || in.Title == "" || in.StimulusType == "" {
		return nil, ErrInvalidInput
	}
	if in.StimulusType != "single" && in.StimulusType != "multiteks" {
		return nil, fmt.Errorf("%w: stimulus_type must be single or multiteks", ErrInvalidInput)
	}
	if len(in.Content) == 0 {
		in.Content = json.RawMessage(`{}`)
	}
	if err := validateStimulusContent(in.StimulusType, in.Content); err != nil {
		return nil, err
	}

	var subjectExists bool
	if err := s.db.QueryRowContext(ctx, `
		SELECT EXISTS(SELECT 1 FROM subjects WHERE id = $1 AND is_active = TRUE)
	`, in.SubjectID).Scan(&subjectExists); err != nil {
		return nil, fmt.Errorf("check subject: %w", err)
	}
	if !subjectExists {
		return nil, ErrSubjectNotFound
	}

	row := s.db.QueryRowContext(ctx, `
		UPDATE stimuli
		SET subject_id = $2,
			title = $3,
			stimulus_type = $4,
			content = $5::jsonb,
			updated_at = now()
		WHERE id = $1 AND is_active = TRUE
		RETURNING id, subject_id, title, stimulus_type, content, is_active, created_by, created_at, updated_at
	`, in.ID, in.SubjectID, in.Title, in.StimulusType, []byte(in.Content))

	var out Stimulus
	var createdBy sql.NullInt64
	if err := row.Scan(
		&out.ID,
		&out.SubjectID,
		&out.Title,
		&out.StimulusType,
		&out.Content,
		&out.IsActive,
		&createdBy,
		&out.CreatedAt,
		&out.UpdatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrStimulusNotFound
		}
		return nil, fmt.Errorf("update stimulus: %w", err)
	}
	if createdBy.Valid {
		out.CreatedBy = &createdBy.Int64
	}
	return &out, nil
}

func (s *Service) DeleteStimulus(ctx context.Context, stimulusID int64) error {
	if stimulusID <= 0 {
		return ErrInvalidInput
	}
	var deletedID int64
	if err := s.db.QueryRowContext(ctx, `
		UPDATE stimuli
		SET is_active = FALSE,
			updated_at = now()
		WHERE id = $1 AND is_active = TRUE
		RETURNING id
	`, stimulusID).Scan(&deletedID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrStimulusNotFound
		}
		return fmt.Errorf("delete stimulus: %w", err)
	}
	return nil
}

func (s *Service) CreateQuestionVersion(ctx context.Context, in CreateQuestionVersionInput) (*QuestionVersion, error) {
	if in.QuestionID <= 0 {
		return nil, ErrInvalidInput
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var currentStem string
	var questionType string
	err = tx.QueryRowContext(ctx, `SELECT stem_html, question_type FROM questions WHERE id = $1`, in.QuestionID).Scan(&currentStem, &questionType)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrQuestionNotFound
		}
		return nil, fmt.Errorf("load question: %w", err)
	}

	base, nextVersion, err := s.loadBaseVersionForCreate(ctx, tx, in.QuestionID, currentStem)
	if err != nil {
		return nil, err
	}

	if in.StimulusID != nil {
		if *in.StimulusID <= 0 {
			return nil, fmt.Errorf("%w: stimulus_id must be positive", ErrInvalidInput)
		}
		base.StimulusID = in.StimulusID
	}
	if in.StemHTML != nil {
		v := strings.TrimSpace(*in.StemHTML)
		if v == "" {
			return nil, fmt.Errorf("%w: stem_html cannot be empty", ErrInvalidInput)
		}
		base.StemHTML = v
	}
	if in.ExplanationHTML != nil {
		v := strings.TrimSpace(*in.ExplanationHTML)
		base.ExplanationHTML = &v
	}
	if in.HintHTML != nil {
		v := strings.TrimSpace(*in.HintHTML)
		base.HintHTML = &v
	}
	if len(in.AnswerKey) > 0 {
		if !json.Valid(in.AnswerKey) {
			return nil, fmt.Errorf("%w: answer_key must be valid json", ErrInvalidInput)
		}
		base.AnswerKey = in.AnswerKey
	}
	if in.DurationSeconds != nil {
		if *in.DurationSeconds < 0 {
			return nil, fmt.Errorf("%w: duration_seconds cannot be negative", ErrInvalidInput)
		}
		base.DurationSeconds = in.DurationSeconds
	}
	if in.Weight != nil {
		if *in.Weight <= 0 {
			return nil, fmt.Errorf("%w: weight must be > 0", ErrInvalidInput)
		}
		base.Weight = *in.Weight
	}
	if in.ChangeNote != nil {
		v := strings.TrimSpace(*in.ChangeNote)
		base.ChangeNote = &v
	}

	if strings.TrimSpace(base.StemHTML) == "" {
		return nil, fmt.Errorf("%w: stem_html is required", ErrInvalidInput)
	}
	if len(base.AnswerKey) == 0 {
		base.AnswerKey = json.RawMessage(`{}`)
	}
	if err := validateAnswerKey(questionType, base.AnswerKey); err != nil {
		return nil, err
	}
	normalizedOptions, correctOptionKeys, err := normalizeAndValidateOptions(questionType, in.Options, base.AnswerKey)
	if err != nil {
		return nil, err
	}

	row := tx.QueryRowContext(ctx, `
		INSERT INTO question_versions (
			question_id, version_no, stimulus_id, stem_html, explanation_html, hint_html,
			answer_key, status, is_public, is_active, duration_seconds, weight, change_note,
			created_by, created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6,
			$7::jsonb, 'draft', FALSE, TRUE, $8, $9, $10,
			NULLIF($11, 0), now()
		)
		RETURNING id, question_id, version_no, stimulus_id, stem_html, explanation_html, hint_html,
			answer_key, status, is_public, is_active, duration_seconds, weight, change_note, created_by, created_at
	`, in.QuestionID, nextVersion, nullInt64Ptr(base.StimulusID), base.StemHTML, nullStringPtr(base.ExplanationHTML), nullStringPtr(base.HintHTML), []byte(base.AnswerKey), nullIntPtr(base.DurationSeconds), base.Weight, nullStringPtr(base.ChangeNote), in.CreatedBy)

	out, err := scanQuestionVersion(row)
	if err != nil {
		return nil, fmt.Errorf("insert question version: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `UPDATE questions SET version = $2, updated_at = now() WHERE id = $1`, in.QuestionID, nextVersion); err != nil {
		return nil, fmt.Errorf("update question current version: %w", err)
	}
	if len(normalizedOptions) > 0 {
		if _, err := tx.ExecContext(ctx, `DELETE FROM question_options WHERE question_id = $1`, in.QuestionID); err != nil {
			return nil, fmt.Errorf("clear question options: %w", err)
		}
		for _, opt := range normalizedOptions {
			isCorrect := correctOptionKeys[opt.OptionKey]
			if _, err := tx.ExecContext(ctx, `
				INSERT INTO question_options (question_id, option_key, option_html, is_correct)
				VALUES ($1, $2, $3, $4)
			`, in.QuestionID, opt.OptionKey, opt.OptionHTML, isCorrect); err != nil {
				return nil, fmt.Errorf("insert question option: %w", err)
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit tx: %w", err)
	}
	return out, nil
}

func (s *Service) FinalizeQuestionVersion(ctx context.Context, questionID int64, versionNo int) (*QuestionVersion, error) {
	if questionID <= 0 || versionNo <= 0 {
		return nil, ErrInvalidInput
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	row := tx.QueryRowContext(ctx, `
		SELECT id, question_id, version_no, stimulus_id, stem_html, explanation_html, hint_html,
			answer_key, status, is_public, is_active, duration_seconds, weight, change_note, created_by, created_at
		FROM question_versions
		WHERE question_id = $1 AND version_no = $2
		FOR UPDATE
	`, questionID, versionNo)

	item, err := scanQuestionVersion(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrVersionNotFound
		}
		return nil, fmt.Errorf("load question version: %w", err)
	}

	var questionType string
	if err := tx.QueryRowContext(ctx, `SELECT question_type FROM questions WHERE id = $1`, questionID).Scan(&questionType); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrQuestionNotFound
		}
		return nil, fmt.Errorf("load question type: %w", err)
	}
	if err := validateAnswerKey(questionType, item.AnswerKey); err != nil {
		return nil, err
	}

	if item.Status != "final" {
		if _, err := tx.ExecContext(ctx, `
			UPDATE question_versions
			SET status = 'final', is_public = TRUE, is_active = TRUE
			WHERE question_id = $1 AND version_no = $2
		`, questionID, versionNo); err != nil {
			return nil, fmt.Errorf("finalize version: %w", err)
		}
		if _, err := tx.ExecContext(ctx, `
			UPDATE question_versions
			SET is_active = FALSE
			WHERE question_id = $1 AND version_no <> $2
		`, questionID, versionNo); err != nil {
			return nil, fmt.Errorf("deactivate other versions: %w", err)
		}
		if _, err := tx.ExecContext(ctx, `UPDATE questions SET version = $2, updated_at = now() WHERE id = $1`, questionID, versionNo); err != nil {
			return nil, fmt.Errorf("update question version pointer: %w", err)
		}
	}

	refetch := tx.QueryRowContext(ctx, `
		SELECT id, question_id, version_no, stimulus_id, stem_html, explanation_html, hint_html,
			answer_key, status, is_public, is_active, duration_seconds, weight, change_note, created_by, created_at
		FROM question_versions
		WHERE question_id = $1 AND version_no = $2
	`, questionID, versionNo)
	out, err := scanQuestionVersion(refetch)
	if err != nil {
		return nil, fmt.Errorf("reload question version: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit tx: %w", err)
	}
	return out, nil
}

func (s *Service) ListQuestionVersions(ctx context.Context, questionID int64) ([]QuestionVersion, error) {
	if questionID <= 0 {
		return nil, ErrInvalidInput
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT id, question_id, version_no, stimulus_id, stem_html, explanation_html, hint_html,
			answer_key, status, is_public, is_active, duration_seconds, weight, change_note, created_by, created_at
		FROM question_versions
		WHERE question_id = $1
		ORDER BY version_no DESC
	`, questionID)
	if err != nil {
		return nil, fmt.Errorf("query question versions: %w", err)
	}
	defer rows.Close()

	items := make([]QuestionVersion, 0)
	for rows.Next() {
		item, err := scanQuestionVersion(rows)
		if err != nil {
			return nil, fmt.Errorf("scan question version: %w", err)
		}
		items = append(items, *item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate question versions: %w", err)
	}
	return items, nil
}

func (s *Service) CreateQuestionParallel(ctx context.Context, in CreateQuestionParallelInput) (*QuestionParallel, error) {
	in.ParallelGroup = strings.TrimSpace(in.ParallelGroup)
	in.ParallelLabel = strings.TrimSpace(in.ParallelLabel)
	if in.ParallelGroup == "" {
		in.ParallelGroup = "default"
	}
	if in.ExamID <= 0 || in.QuestionID <= 0 || in.ParallelOrder <= 0 || in.ParallelLabel == "" {
		return nil, ErrInvalidInput
	}

	if err := s.ensureExamAndQuestionLinked(ctx, in.ExamID, in.QuestionID); err != nil {
		return nil, err
	}

	row := s.db.QueryRowContext(ctx, `
		INSERT INTO question_parallels (exam_id, question_id, parallel_group, parallel_order, parallel_label, is_active, created_at)
		VALUES ($1, $2, $3, $4, $5, TRUE, now())
		RETURNING id, exam_id, question_id, parallel_group, parallel_order, parallel_label, is_active, created_at
	`, in.ExamID, in.QuestionID, in.ParallelGroup, in.ParallelOrder, in.ParallelLabel)
	out, err := scanQuestionParallel(row)
	if err != nil {
		return nil, fmt.Errorf("insert question parallel: %w", err)
	}
	return out, nil
}

func (s *Service) ListQuestionParallels(ctx context.Context, examID int64, parallelGroup string) ([]QuestionParallel, error) {
	if examID <= 0 {
		return nil, ErrInvalidInput
	}
	parallelGroup = strings.TrimSpace(parallelGroup)

	var examExists bool
	if err := s.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM exams WHERE id = $1)`, examID).Scan(&examExists); err != nil {
		return nil, fmt.Errorf("check exam exists: %w", err)
	}
	if !examExists {
		return nil, ErrExamNotFound
	}

	query := `
		SELECT id, exam_id, question_id, parallel_group, parallel_order, parallel_label, is_active, created_at
		FROM question_parallels
		WHERE exam_id = $1
	`
	args := []any{examID}
	if parallelGroup != "" {
		query += ` AND parallel_group = $2`
		args = append(args, parallelGroup)
	}
	query += ` ORDER BY parallel_group ASC, parallel_order ASC, id ASC`

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query question parallels: %w", err)
	}
	defer rows.Close()

	items := make([]QuestionParallel, 0)
	for rows.Next() {
		item, err := scanQuestionParallel(rows)
		if err != nil {
			return nil, fmt.Errorf("scan question parallel: %w", err)
		}
		items = append(items, *item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate question parallels: %w", err)
	}
	return items, nil
}

func (s *Service) UpdateQuestionParallel(ctx context.Context, in UpdateQuestionParallelInput) (*QuestionParallel, error) {
	in.ParallelGroup = strings.TrimSpace(in.ParallelGroup)
	in.ParallelLabel = strings.TrimSpace(in.ParallelLabel)
	if in.ParallelGroup == "" {
		in.ParallelGroup = "default"
	}
	if in.ExamID <= 0 || in.ParallelID <= 0 || in.QuestionID <= 0 || in.ParallelOrder <= 0 || in.ParallelLabel == "" {
		return nil, ErrInvalidInput
	}

	if err := s.ensureExamAndQuestionLinked(ctx, in.ExamID, in.QuestionID); err != nil {
		return nil, err
	}

	row := s.db.QueryRowContext(ctx, `
		UPDATE question_parallels
		SET question_id = $3,
			parallel_group = $4,
			parallel_order = $5,
			parallel_label = $6,
			is_active = TRUE
		WHERE id = $1 AND exam_id = $2
		RETURNING id, exam_id, question_id, parallel_group, parallel_order, parallel_label, is_active, created_at
	`, in.ParallelID, in.ExamID, in.QuestionID, in.ParallelGroup, in.ParallelOrder, in.ParallelLabel)
	out, err := scanQuestionParallel(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrParallelNotFound
		}
		return nil, fmt.Errorf("update question parallel: %w", err)
	}
	return out, nil
}

func (s *Service) DeleteQuestionParallel(ctx context.Context, examID, parallelID int64) error {
	if examID <= 0 || parallelID <= 0 {
		return ErrInvalidInput
	}
	res, err := s.db.ExecContext(ctx, `DELETE FROM question_parallels WHERE id = $1 AND exam_id = $2`, parallelID, examID)
	if err != nil {
		return fmt.Errorf("delete question parallel: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete question parallel affected rows: %w", err)
	}
	if affected == 0 {
		return ErrParallelNotFound
	}
	return nil
}

func (s *Service) CreateReviewTask(ctx context.Context, in CreateReviewTaskInput) (*ReviewTask, error) {
	in.Note = strings.TrimSpace(in.Note)
	if in.QuestionVersionID <= 0 || in.ReviewerID <= 0 {
		return nil, ErrInvalidInput
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var questionID int64
	err = tx.QueryRowContext(ctx, `
		SELECT question_id FROM question_versions WHERE id = $1
	`, in.QuestionVersionID).Scan(&questionID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrVersionNotFound
		}
		return nil, fmt.Errorf("load question version: %w", err)
	}

	if in.ExamID != nil && *in.ExamID > 0 {
		var examExists bool
		if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM exams WHERE id = $1)`, *in.ExamID).Scan(&examExists); err != nil {
			return nil, fmt.Errorf("check exam exists: %w", err)
		}
		if !examExists {
			return nil, ErrExamNotFound
		}
	}

	row := tx.QueryRowContext(ctx, `
		INSERT INTO review_tasks (question_version_id, exam_id, reviewer_id, status, assigned_at, note)
		VALUES ($1, $2, $3, 'menunggu_reviu', now(), NULLIF($4,''))
		ON CONFLICT (question_version_id, reviewer_id)
		DO UPDATE SET status='menunggu_reviu', assigned_at=now(), note=EXCLUDED.note
		RETURNING id, question_version_id, reviewer_id, status, assigned_at, reviewed_at, note, exam_id
	`, in.QuestionVersionID, nullInt64Ptr(in.ExamID), in.ReviewerID, in.Note)

	var out ReviewTask
	var reviewedAt sql.NullTime
	var note sql.NullString
	var examID sql.NullInt64
	if err := row.Scan(&out.ID, &out.QuestionVersionID, &out.ReviewerID, &out.Status, &out.AssignedAt, &reviewedAt, &note, &examID); err != nil {
		return nil, fmt.Errorf("insert review task: %w", err)
	}
	out.QuestionID = questionID
	if reviewedAt.Valid {
		out.ReviewedAt = &reviewedAt.Time
	}
	if note.Valid {
		out.Note = &note.String
	}
	if examID.Valid {
		out.ExamID = &examID.Int64
	}

	if _, err := tx.ExecContext(ctx, `
		UPDATE question_versions
		SET status = 'menunggu_reviu'
		WHERE id = $1
	`, in.QuestionVersionID); err != nil {
		return nil, fmt.Errorf("update question_version review status: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit tx: %w", err)
	}
	return &out, nil
}

func (s *Service) DecideReviewTask(ctx context.Context, in ReviewDecisionInput) (*ReviewTask, error) {
	in.Status = strings.TrimSpace(in.Status)
	in.Note = strings.TrimSpace(in.Note)
	if in.TaskID <= 0 || in.ReviewerID <= 0 {
		return nil, ErrInvalidInput
	}
	if in.Status != "disetujui" && in.Status != "perlu_revisi" {
		return nil, ErrInvalidInput
	}
	if in.Status == "perlu_revisi" && in.Note == "" {
		return nil, fmt.Errorf("%w: review note is required for perlu_revisi", ErrInvalidInput)
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var questionVersionID int64
	var questionID int64
	var reviewerID int64
	err = tx.QueryRowContext(ctx, `
		SELECT rt.question_version_id, qv.question_id, rt.reviewer_id
		FROM review_tasks rt
		JOIN question_versions qv ON qv.id = rt.question_version_id
		WHERE rt.id = $1
		FOR UPDATE
	`, in.TaskID).Scan(&questionVersionID, &questionID, &reviewerID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrReviewTaskNotFound
		}
		return nil, fmt.Errorf("load review task: %w", err)
	}
	if !in.IsPrivileged && reviewerID != in.ReviewerID {
		return nil, ErrReviewForbidden
	}

	_, err = tx.ExecContext(ctx, `
		UPDATE review_tasks
		SET status = $2, reviewed_at = now(), note = NULLIF($3,'')
		WHERE id = $1
	`, in.TaskID, in.Status, in.Note)
	if err != nil {
		return nil, fmt.Errorf("update review task decision: %w", err)
	}

	commentType := "keputusan"
	if in.Status == "perlu_revisi" {
		commentType = "revisi"
	}
	if in.Note != "" {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO review_comments (review_task_id, author_id, comment_type, comment_text, created_at)
			VALUES ($1, $2, $3, $4, now())
		`, in.TaskID, in.ReviewerID, commentType, in.Note); err != nil {
			return nil, fmt.Errorf("insert review comment: %w", err)
		}
	}

	nextQvStatus := "revisi"
	if in.Status == "disetujui" {
		nextQvStatus = "final"
	}
	if _, err := tx.ExecContext(ctx, `
		UPDATE question_versions
		SET status = $2
		WHERE id = $1
	`, questionVersionID, nextQvStatus); err != nil {
		return nil, fmt.Errorf("update question_version status from review: %w", err)
	}

	row := tx.QueryRowContext(ctx, `
		SELECT id, question_version_id, reviewer_id, status, assigned_at, reviewed_at, note, exam_id
		FROM review_tasks
		WHERE id = $1
	`, in.TaskID)
	var out ReviewTask
	var reviewedAt sql.NullTime
	var note sql.NullString
	var examID sql.NullInt64
	if err := row.Scan(&out.ID, &out.QuestionVersionID, &out.ReviewerID, &out.Status, &out.AssignedAt, &reviewedAt, &note, &examID); err != nil {
		return nil, fmt.Errorf("reload review task: %w", err)
	}
	out.QuestionID = questionID
	if reviewedAt.Valid {
		out.ReviewedAt = &reviewedAt.Time
	}
	if note.Valid {
		out.Note = &note.String
	}
	if examID.Valid {
		out.ExamID = &examID.Int64
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit tx: %w", err)
	}
	return &out, nil
}

func (s *Service) ListReviewTasks(ctx context.Context, reviewerID int64, status string) ([]ReviewTask, error) {
	status = strings.TrimSpace(status)
	if reviewerID <= 0 {
		return nil, ErrInvalidInput
	}
	if status != "" && status != "menunggu_reviu" && status != "disetujui" && status != "perlu_revisi" {
		return nil, ErrInvalidInput
	}

	query := `
		SELECT rt.id, rt.question_version_id, qv.question_id, rt.reviewer_id, rt.status, rt.assigned_at, rt.reviewed_at, rt.note, rt.exam_id
		FROM review_tasks rt
		JOIN question_versions qv ON qv.id = rt.question_version_id
		WHERE rt.reviewer_id = $1
	`
	args := []any{reviewerID}
	if status != "" {
		query += ` AND rt.status = $2`
		args = append(args, status)
	}
	query += ` ORDER BY rt.assigned_at DESC`

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query review tasks: %w", err)
	}
	defer rows.Close()

	items := make([]ReviewTask, 0)
	for rows.Next() {
		var item ReviewTask
		var reviewedAt sql.NullTime
		var note sql.NullString
		var examID sql.NullInt64
		if err := rows.Scan(&item.ID, &item.QuestionVersionID, &item.QuestionID, &item.ReviewerID, &item.Status, &item.AssignedAt, &reviewedAt, &note, &examID); err != nil {
			return nil, fmt.Errorf("scan review task: %w", err)
		}
		if reviewedAt.Valid {
			item.ReviewedAt = &reviewedAt.Time
		}
		if note.Valid {
			item.Note = &note.String
		}
		if examID.Valid {
			item.ExamID = &examID.Int64
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate review tasks: %w", err)
	}
	return items, nil
}

func (s *Service) GetQuestionReviews(ctx context.Context, questionID int64) ([]QuestionReview, error) {
	if questionID <= 0 {
		return nil, ErrInvalidInput
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT rt.id, rt.question_version_id, qv.question_id, rt.reviewer_id, rt.status, rt.assigned_at, rt.reviewed_at, rt.note, rt.exam_id
		FROM review_tasks rt
		JOIN question_versions qv ON qv.id = rt.question_version_id
		WHERE qv.question_id = $1
		ORDER BY rt.assigned_at DESC
	`, questionID)
	if err != nil {
		return nil, fmt.Errorf("query question review tasks: %w", err)
	}
	defer rows.Close()

	out := make([]QuestionReview, 0)
	for rows.Next() {
		var task ReviewTask
		var reviewedAt sql.NullTime
		var note sql.NullString
		var examID sql.NullInt64
		if err := rows.Scan(&task.ID, &task.QuestionVersionID, &task.QuestionID, &task.ReviewerID, &task.Status, &task.AssignedAt, &reviewedAt, &note, &examID); err != nil {
			return nil, fmt.Errorf("scan question review task: %w", err)
		}
		if reviewedAt.Valid {
			task.ReviewedAt = &reviewedAt.Time
		}
		if note.Valid {
			task.Note = &note.String
		}
		if examID.Valid {
			task.ExamID = &examID.Int64
		}

		comments, err := s.listReviewComments(ctx, task.ID)
		if err != nil {
			return nil, err
		}
		out = append(out, QuestionReview{
			Task:     task,
			Comments: comments,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate question review tasks: %w", err)
	}
	return out, nil
}

func (s *Service) listReviewComments(ctx context.Context, taskID int64) ([]ReviewComment, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, review_task_id, author_id, comment_type, comment_text, created_at
		FROM review_comments
		WHERE review_task_id = $1
		ORDER BY created_at ASC, id ASC
	`, taskID)
	if err != nil {
		return nil, fmt.Errorf("query review comments: %w", err)
	}
	defer rows.Close()

	items := make([]ReviewComment, 0)
	for rows.Next() {
		var c ReviewComment
		if err := rows.Scan(&c.ID, &c.ReviewTaskID, &c.AuthorID, &c.CommentType, &c.CommentText, &c.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan review comment: %w", err)
		}
		items = append(items, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate review comments: %w", err)
	}
	return items, nil
}

func (s *Service) ensureExamAndQuestionLinked(ctx context.Context, examID, questionID int64) error {
	var examExists bool
	if err := s.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM exams WHERE id = $1)`, examID).Scan(&examExists); err != nil {
		return fmt.Errorf("check exam exists: %w", err)
	}
	if !examExists {
		return ErrExamNotFound
	}

	var linked bool
	if err := s.db.QueryRowContext(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM exam_questions WHERE exam_id = $1 AND question_id = $2
		)
	`, examID, questionID).Scan(&linked); err != nil {
		return fmt.Errorf("check question in exam: %w", err)
	}
	if !linked {
		return ErrQuestionNotInExam
	}
	return nil
}

type baseVersionData struct {
	StimulusID      *int64
	StemHTML        string
	ExplanationHTML *string
	HintHTML        *string
	AnswerKey       json.RawMessage
	DurationSeconds *int
	Weight          float64
	ChangeNote      *string
}

func (s *Service) loadBaseVersionForCreate(ctx context.Context, tx *sql.Tx, questionID int64, defaultStem string) (*baseVersionData, int, error) {
	row := tx.QueryRowContext(ctx, `
		SELECT version_no, stimulus_id, stem_html, explanation_html, hint_html,
			answer_key, duration_seconds, weight, change_note
		FROM question_versions
		WHERE question_id = $1
		ORDER BY version_no DESC
		LIMIT 1
		FOR UPDATE
	`, questionID)

	var versionNo int
	var stimulusID sql.NullInt64
	var stemHTML string
	var explanation sql.NullString
	var hint sql.NullString
	var answerKey json.RawMessage
	var duration sql.NullInt64
	var weight float64
	var change sql.NullString
	if err := row.Scan(&versionNo, &stimulusID, &stemHTML, &explanation, &hint, &answerKey, &duration, &weight, &change); err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return nil, 0, fmt.Errorf("load latest version: %w", err)
		}

		if strings.TrimSpace(defaultStem) == "" {
			return nil, 0, fmt.Errorf("%w: question stem_html is empty", ErrInvalidInput)
		}
		return &baseVersionData{
			StemHTML:  defaultStem,
			AnswerKey: json.RawMessage(`{}`),
			Weight:    1,
		}, 1, nil
	}

	var stimulusPtr *int64
	if stimulusID.Valid {
		stimulusPtr = &stimulusID.Int64
	}
	var explanationPtr *string
	if explanation.Valid {
		explanationPtr = &explanation.String
	}
	var hintPtr *string
	if hint.Valid {
		hintPtr = &hint.String
	}
	var durationPtr *int
	if duration.Valid {
		d := int(duration.Int64)
		durationPtr = &d
	}
	var changePtr *string
	if change.Valid {
		changePtr = &change.String
	}
	if len(answerKey) == 0 {
		answerKey = json.RawMessage(`{}`)
	}
	if weight <= 0 {
		weight = 1
	}

	return &baseVersionData{
		StimulusID:      stimulusPtr,
		StemHTML:        stemHTML,
		ExplanationHTML: explanationPtr,
		HintHTML:        hintPtr,
		AnswerKey:       answerKey,
		DurationSeconds: durationPtr,
		Weight:          weight,
		ChangeNote:      changePtr,
	}, versionNo + 1, nil
}

func validateStimulusContent(stimulusType string, raw json.RawMessage) error {
	var content map[string]any
	if err := json.Unmarshal(raw, &content); err != nil {
		return fmt.Errorf("%w: content must be valid JSON object", ErrInvalidInput)
	}

	switch stimulusType {
	case "single":
		body, ok := content["body"].(string)
		if !ok || strings.TrimSpace(body) == "" {
			return fmt.Errorf("%w: single content.body is required", ErrInvalidInput)
		}
	case "multiteks":
		rawTabs, ok := content["tabs"]
		if !ok {
			return fmt.Errorf("%w: multiteks content.tabs is required", ErrInvalidInput)
		}
		tabs, ok := rawTabs.([]any)
		if !ok || len(tabs) == 0 {
			return fmt.Errorf("%w: multiteks content.tabs must be non-empty array", ErrInvalidInput)
		}
		for i, tabRaw := range tabs {
			tab, ok := tabRaw.(map[string]any)
			if !ok {
				return fmt.Errorf("%w: tabs[%d] must be object", ErrInvalidInput, i)
			}
			title, _ := tab["title"].(string)
			body, _ := tab["body"].(string)
			if strings.TrimSpace(title) == "" || strings.TrimSpace(body) == "" {
				return fmt.Errorf("%w: tabs[%d].title and tabs[%d].body are required", ErrInvalidInput, i, i)
			}
		}
	}
	return nil
}

func validateAnswerKey(questionType string, raw json.RawMessage) error {
	questionType = strings.TrimSpace(strings.ToLower(questionType))
	if len(raw) == 0 || !json.Valid(raw) {
		return fmt.Errorf("%w: answer_key must be valid json", ErrInvalidInput)
	}

	var obj map[string]any
	if err := json.Unmarshal(raw, &obj); err != nil {
		return fmt.Errorf("%w: answer_key must be object", ErrInvalidInput)
	}

	switch questionType {
	case "pg_tunggal":
		correct, ok := obj["correct"].(string)
		if !ok || strings.TrimSpace(correct) == "" {
			return fmt.Errorf("%w: pg_tunggal answer_key.correct (string) is required", ErrInvalidInput)
		}
	case "multi_jawaban":
		correct, ok := obj["correct"].([]any)
		if !ok || len(correct) == 0 {
			return fmt.Errorf("%w: multi_jawaban answer_key.correct (non-empty array) is required", ErrInvalidInput)
		}
		seen := map[string]struct{}{}
		for i, item := range correct {
			v, ok := item.(string)
			v = strings.TrimSpace(v)
			if !ok || v == "" {
				return fmt.Errorf("%w: multi_jawaban answer_key.correct[%d] must be non-empty string", ErrInvalidInput, i)
			}
			if _, exists := seen[v]; exists {
				return fmt.Errorf("%w: multi_jawaban answer_key.correct has duplicate '%s'", ErrInvalidInput, v)
			}
			seen[v] = struct{}{}
		}
		if modeRaw, ok := obj["mode"]; ok {
			mode, ok := modeRaw.(string)
			mode = strings.TrimSpace(strings.ToLower(mode))
			if !ok || mode == "" {
				return fmt.Errorf("%w: multi_jawaban answer_key.mode must be string", ErrInvalidInput)
			}
			if mode != "exact" {
				return fmt.Errorf("%w: multi_jawaban answer_key.mode must be 'exact'", ErrInvalidInput)
			}
		}
	case "benar_salah_pernyataan":
		statements, ok := obj["statements"].([]any)
		if !ok || len(statements) == 0 {
			return fmt.Errorf("%w: benar_salah_pernyataan answer_key.statements (non-empty array) is required", ErrInvalidInput)
		}
		seen := map[string]struct{}{}
		for i, item := range statements {
			stmt, ok := item.(map[string]any)
			if !ok {
				return fmt.Errorf("%w: statements[%d] must be object", ErrInvalidInput, i)
			}
			id, _ := stmt["id"].(string)
			id = strings.TrimSpace(id)
			correct, ok := stmt["correct"].(bool)
			_ = correct
			if id == "" || !ok {
				return fmt.Errorf("%w: statements[%d].id and statements[%d].correct are required", ErrInvalidInput, i, i)
			}
			if _, exists := seen[id]; exists {
				return fmt.Errorf("%w: duplicate statement id '%s'", ErrInvalidInput, id)
			}
			seen[id] = struct{}{}
		}
	default:
		return fmt.Errorf("%w: unsupported question_type '%s'", ErrInvalidInput, questionType)
	}

	return nil
}

func parseCorrectKeysForOptions(questionType string, raw json.RawMessage) (map[string]bool, error) {
	keys := map[string]bool{}
	var obj map[string]any
	if err := json.Unmarshal(raw, &obj); err != nil {
		return nil, fmt.Errorf("%w: answer_key must be object", ErrInvalidInput)
	}
	switch strings.TrimSpace(strings.ToLower(questionType)) {
	case "pg_tunggal":
		v, _ := obj["correct"].(string)
		v = strings.TrimSpace(strings.ToUpper(v))
		if v == "" {
			return nil, fmt.Errorf("%w: pg_tunggal answer_key.correct (string) is required", ErrInvalidInput)
		}
		keys[v] = true
	case "multi_jawaban":
		rawKeys, _ := obj["correct"].([]any)
		if len(rawKeys) == 0 {
			return nil, fmt.Errorf("%w: multi_jawaban answer_key.correct (non-empty array) is required", ErrInvalidInput)
		}
		for i, item := range rawKeys {
			v, _ := item.(string)
			v = strings.TrimSpace(strings.ToUpper(v))
			if v == "" {
				return nil, fmt.Errorf("%w: multi_jawaban answer_key.correct[%d] must be non-empty string", ErrInvalidInput, i)
			}
			keys[v] = true
		}
	}
	return keys, nil
}

func normalizeAndValidateOptions(questionType string, options []QuestionOptionInput, answerKey json.RawMessage) ([]QuestionOptionInput, map[string]bool, error) {
	qType := strings.TrimSpace(strings.ToLower(questionType))
	if qType == "benar_salah_pernyataan" {
		return nil, nil, nil
	}
	if len(options) == 0 {
		return nil, nil, nil
	}

	correctKeys, err := parseCorrectKeysForOptions(qType, answerKey)
	if err != nil {
		return nil, nil, err
	}
	if len(options) < 2 {
		return nil, nil, fmt.Errorf("%w: options must contain at least 2 rows", ErrInvalidInput)
	}

	seen := map[string]struct{}{}
	out := make([]QuestionOptionInput, 0, len(options))
	for i, it := range options {
		key := strings.TrimSpace(strings.ToUpper(it.OptionKey))
		html := strings.TrimSpace(it.OptionHTML)
		if key == "" {
			return nil, nil, fmt.Errorf("%w: options[%d].option_key is required", ErrInvalidInput, i)
		}
		if html == "" {
			return nil, nil, fmt.Errorf("%w: options[%d].option_html is required", ErrInvalidInput, i)
		}
		if _, ok := seen[key]; ok {
			return nil, nil, fmt.Errorf("%w: duplicate option_key '%s'", ErrInvalidInput, key)
		}
		seen[key] = struct{}{}
		out = append(out, QuestionOptionInput{
			OptionKey:  key,
			OptionHTML: html,
		})
	}

	for key := range correctKeys {
		if _, ok := seen[key]; !ok {
			return nil, nil, fmt.Errorf("%w: answer_key references unknown option '%s'", ErrInvalidInput, key)
		}
	}
	return out, correctKeys, nil
}

func scanQuestionVersion(scanner interface{ Scan(dest ...any) error }) (*QuestionVersion, error) {
	var out QuestionVersion
	var stimulusID sql.NullInt64
	var explanation sql.NullString
	var hint sql.NullString
	var duration sql.NullInt64
	var change sql.NullString
	var createdBy sql.NullInt64
	if err := scanner.Scan(
		&out.ID,
		&out.QuestionID,
		&out.VersionNo,
		&stimulusID,
		&out.StemHTML,
		&explanation,
		&hint,
		&out.AnswerKey,
		&out.Status,
		&out.IsPublic,
		&out.IsActive,
		&duration,
		&out.Weight,
		&change,
		&createdBy,
		&out.CreatedAt,
	); err != nil {
		return nil, err
	}
	if stimulusID.Valid {
		out.StimulusID = &stimulusID.Int64
	}
	if explanation.Valid {
		out.ExplanationHTML = &explanation.String
	}
	if hint.Valid {
		out.HintHTML = &hint.String
	}
	if duration.Valid {
		d := int(duration.Int64)
		out.DurationSeconds = &d
	}
	if change.Valid {
		out.ChangeNote = &change.String
	}
	if createdBy.Valid {
		out.CreatedBy = &createdBy.Int64
	}
	return &out, nil
}

func scanQuestionParallel(scanner interface{ Scan(dest ...any) error }) (*QuestionParallel, error) {
	var out QuestionParallel
	if err := scanner.Scan(
		&out.ID,
		&out.ExamID,
		&out.QuestionID,
		&out.ParallelGroup,
		&out.ParallelOrder,
		&out.ParallelLabel,
		&out.IsActive,
		&out.CreatedAt,
	); err != nil {
		return nil, err
	}
	return &out, nil
}

func nullStringPtr(v *string) any {
	if v == nil {
		return nil
	}
	s := strings.TrimSpace(*v)
	if s == "" {
		return nil
	}
	return s
}

func nullInt64Ptr(v *int64) any {
	if v == nil {
		return nil
	}
	if *v <= 0 {
		return nil
	}
	return *v
}

func nullIntPtr(v *int) any {
	if v == nil {
		return nil
	}
	if *v < 0 {
		return nil
	}
	return *v
}
