CREATE TABLE IF NOT EXISTS exam_question_incidents (
  id BIGSERIAL PRIMARY KEY,
  exam_id BIGINT NOT NULL REFERENCES exams(id) ON DELETE CASCADE,
  question_id BIGINT NOT NULL REFERENCES questions(id) ON DELETE CASCADE,
  policy VARCHAR(20) NOT NULL CHECK (policy IN ('drop','bonus_all')),
  reason TEXT NOT NULL,
  is_active BOOLEAN NOT NULL DEFAULT TRUE,
  decided_by BIGINT REFERENCES users(id),
  decided_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (exam_id, question_id)
);

CREATE INDEX IF NOT EXISTS idx_exam_question_incidents_exam_active
  ON exam_question_incidents (exam_id, is_active, question_id);
