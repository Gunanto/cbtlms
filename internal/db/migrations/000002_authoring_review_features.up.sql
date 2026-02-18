BEGIN;

CREATE TABLE stimuli (
  id BIGSERIAL PRIMARY KEY,
  subject_id BIGINT NOT NULL REFERENCES subjects(id),
  title VARCHAR(200) NOT NULL,
  stimulus_type VARCHAR(20) NOT NULL CHECK (stimulus_type IN ('single','multiteks')),
  content JSONB NOT NULL DEFAULT '{}'::jsonb,
  is_active BOOLEAN NOT NULL DEFAULT TRUE,
  created_by BIGINT REFERENCES users(id),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE question_versions (
  id BIGSERIAL PRIMARY KEY,
  question_id BIGINT NOT NULL REFERENCES questions(id) ON DELETE CASCADE,
  version_no INT NOT NULL,
  stimulus_id BIGINT REFERENCES stimuli(id),
  stem_html TEXT NOT NULL,
  explanation_html TEXT,
  hint_html TEXT,
  answer_key JSONB NOT NULL DEFAULT '{}'::jsonb,
  status VARCHAR(30) NOT NULL CHECK (status IN ('draft','final','menunggu_reviu','revisi')),
  is_public BOOLEAN NOT NULL DEFAULT FALSE,
  is_active BOOLEAN NOT NULL DEFAULT TRUE,
  duration_seconds INT,
  weight NUMERIC(6,2) NOT NULL DEFAULT 1.0,
  change_note TEXT,
  created_by BIGINT REFERENCES users(id),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (question_id, version_no)
);

CREATE TABLE question_parallels (
  id BIGSERIAL PRIMARY KEY,
  exam_id BIGINT NOT NULL REFERENCES exams(id) ON DELETE CASCADE,
  question_id BIGINT NOT NULL REFERENCES questions(id),
  parallel_group VARCHAR(50) NOT NULL DEFAULT 'default',
  parallel_order INT NOT NULL,
  parallel_label VARCHAR(30) NOT NULL,
  is_active BOOLEAN NOT NULL DEFAULT TRUE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (exam_id, parallel_group, parallel_order),
  UNIQUE (exam_id, question_id, parallel_label)
);

CREATE TABLE review_tasks (
  id BIGSERIAL PRIMARY KEY,
  question_version_id BIGINT NOT NULL REFERENCES question_versions(id) ON DELETE CASCADE,
  exam_id BIGINT REFERENCES exams(id) ON DELETE SET NULL,
  reviewer_id BIGINT NOT NULL REFERENCES users(id),
  status VARCHAR(30) NOT NULL CHECK (status IN ('menunggu_reviu','disetujui','perlu_revisi')),
  assigned_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  reviewed_at TIMESTAMPTZ,
  note TEXT,
  UNIQUE (question_version_id, reviewer_id)
);

CREATE TABLE review_comments (
  id BIGSERIAL PRIMARY KEY,
  review_task_id BIGINT NOT NULL REFERENCES review_tasks(id) ON DELETE CASCADE,
  author_id BIGINT NOT NULL REFERENCES users(id),
  comment_type VARCHAR(20) NOT NULL CHECK (comment_type IN ('catatan','revisi','keputusan')),
  comment_text TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_stimuli_subject ON stimuli (subject_id);
CREATE INDEX idx_qv_question_status ON question_versions (question_id, status);
CREATE INDEX idx_qp_exam_group ON question_parallels (exam_id, parallel_group);
CREATE INDEX idx_rt_reviewer_status ON review_tasks (reviewer_id, status);
CREATE INDEX idx_rc_review_task ON review_comments (review_task_id);

COMMIT;
