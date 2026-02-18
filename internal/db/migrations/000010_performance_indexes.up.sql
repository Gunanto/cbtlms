BEGIN;

-- Speeds up scoring query that fetches only correct options per question.
CREATE INDEX IF NOT EXISTS idx_question_options_question_correct
  ON question_options (question_id, is_correct);

-- Speeds up attempt progress counters and doubt toggles.
CREATE INDEX IF NOT EXISTS idx_attempt_answers_attempt_doubt
  ON attempt_answers (attempt_id, is_doubt);

-- Helps operational queries by student/status and timeout checks.
CREATE INDEX IF NOT EXISTS idx_attempts_student_status_expires
  ON attempts (student_id, status, expires_at);

-- Helps review workflow lookups by question version status.
CREATE INDEX IF NOT EXISTS idx_review_tasks_qv_status
  ON review_tasks (question_version_id, status);

COMMIT;
