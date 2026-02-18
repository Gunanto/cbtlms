BEGIN;

DROP INDEX IF EXISTS idx_review_tasks_qv_status;
DROP INDEX IF EXISTS idx_attempts_student_status_expires;
DROP INDEX IF EXISTS idx_attempt_answers_attempt_doubt;
DROP INDEX IF EXISTS idx_question_options_question_correct;

COMMIT;
