BEGIN;

CREATE TABLE users (
  id BIGSERIAL PRIMARY KEY,
  username VARCHAR(100) UNIQUE NOT NULL,
  password_hash TEXT NOT NULL,
  full_name VARCHAR(150) NOT NULL,
  role VARCHAR(20) NOT NULL CHECK (role IN ('admin','proktor','guru','siswa')),
  is_active BOOLEAN NOT NULL DEFAULT TRUE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE student_profiles (
  user_id BIGINT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  nisn VARCHAR(30),
  school_id BIGINT,
  grade_level VARCHAR(20),
  class_name VARCHAR(30)
);

CREATE TABLE subjects (
  id BIGSERIAL PRIMARY KEY,
  education_level VARCHAR(30) NOT NULL,
  subject_type VARCHAR(30) NOT NULL,
  name VARCHAR(100) NOT NULL,
  is_active BOOLEAN NOT NULL DEFAULT TRUE
);

CREATE TABLE questions (
  id BIGSERIAL PRIMARY KEY,
  subject_id BIGINT NOT NULL REFERENCES subjects(id),
  question_type VARCHAR(30) NOT NULL,
  stem_html TEXT NOT NULL,
  stimulus_html TEXT,
  difficulty SMALLINT,
  metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
  is_active BOOLEAN NOT NULL DEFAULT TRUE,
  version INT NOT NULL DEFAULT 1,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE question_options (
  id BIGSERIAL PRIMARY KEY,
  question_id BIGINT NOT NULL REFERENCES questions(id) ON DELETE CASCADE,
  option_key VARCHAR(10) NOT NULL,
  option_html TEXT NOT NULL,
  is_correct BOOLEAN NOT NULL DEFAULT FALSE,
  UNIQUE (question_id, option_key)
);

CREATE TABLE exams (
  id BIGSERIAL PRIMARY KEY,
  code VARCHAR(50) UNIQUE NOT NULL,
  title VARCHAR(200) NOT NULL,
  subject_id BIGINT NOT NULL REFERENCES subjects(id),
  duration_minutes INT NOT NULL,
  start_at TIMESTAMPTZ,
  end_at TIMESTAMPTZ,
  randomize_questions BOOLEAN NOT NULL DEFAULT FALSE,
  randomize_options BOOLEAN NOT NULL DEFAULT FALSE,
  review_policy VARCHAR(30) NOT NULL DEFAULT 'after_submit',
  is_active BOOLEAN NOT NULL DEFAULT TRUE,
  created_by BIGINT REFERENCES users(id),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE exam_questions (
  exam_id BIGINT NOT NULL REFERENCES exams(id) ON DELETE CASCADE,
  question_id BIGINT NOT NULL REFERENCES questions(id),
  seq_no INT NOT NULL,
  weight NUMERIC(6,2) NOT NULL DEFAULT 1.0,
  PRIMARY KEY (exam_id, question_id),
  UNIQUE (exam_id, seq_no)
);

CREATE TABLE attempts (
  id BIGSERIAL PRIMARY KEY,
  exam_id BIGINT NOT NULL REFERENCES exams(id),
  student_id BIGINT NOT NULL REFERENCES users(id),
  status VARCHAR(20) NOT NULL CHECK (status IN ('in_progress','submitted','expired','cancelled')),
  started_at TIMESTAMPTZ NOT NULL,
  expires_at TIMESTAMPTZ NOT NULL,
  submitted_at TIMESTAMPTZ,
  score NUMERIC(7,2),
  total_correct INT,
  total_wrong INT,
  total_unanswered INT,
  client_meta JSONB NOT NULL DEFAULT '{}'::jsonb,
  UNIQUE (exam_id, student_id)
);

CREATE TABLE attempt_answers (
  id BIGSERIAL PRIMARY KEY,
  attempt_id BIGINT NOT NULL REFERENCES attempts(id) ON DELETE CASCADE,
  question_id BIGINT NOT NULL REFERENCES questions(id),
  answer_payload JSONB,
  is_doubt BOOLEAN NOT NULL DEFAULT FALSE,
  is_final BOOLEAN NOT NULL DEFAULT FALSE,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (attempt_id, question_id)
);

CREATE TABLE attempt_scores (
  attempt_id BIGINT NOT NULL REFERENCES attempts(id) ON DELETE CASCADE,
  question_id BIGINT NOT NULL REFERENCES questions(id),
  earned_score NUMERIC(6,2) NOT NULL,
  is_correct BOOLEAN,
  feedback JSONB NOT NULL DEFAULT '{}'::jsonb,
  PRIMARY KEY (attempt_id, question_id)
);

CREATE TABLE audit_logs (
  id BIGSERIAL PRIMARY KEY,
  user_id BIGINT REFERENCES users(id),
  action VARCHAR(100) NOT NULL,
  entity_type VARCHAR(50),
  entity_id VARCHAR(50),
  payload JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_attempts_exam_status ON attempts (exam_id, status);
CREATE INDEX idx_attempt_answers_attempt ON attempt_answers (attempt_id);
CREATE INDEX idx_questions_subject ON questions (subject_id);

COMMIT;
