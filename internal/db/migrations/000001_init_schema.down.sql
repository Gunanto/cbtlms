BEGIN;

DROP INDEX IF EXISTS idx_questions_subject;
DROP INDEX IF EXISTS idx_attempt_answers_attempt;
DROP INDEX IF EXISTS idx_attempts_exam_status;

DROP TABLE IF EXISTS audit_logs;
DROP TABLE IF EXISTS attempt_scores;
DROP TABLE IF EXISTS attempt_answers;
DROP TABLE IF EXISTS attempts;
DROP TABLE IF EXISTS exam_questions;
DROP TABLE IF EXISTS exams;
DROP TABLE IF EXISTS question_options;
DROP TABLE IF EXISTS questions;
DROP TABLE IF EXISTS subjects;
DROP TABLE IF EXISTS student_profiles;
DROP TABLE IF EXISTS users;

COMMIT;
