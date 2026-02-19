CREATE TABLE IF NOT EXISTS exam_assignments (
  id BIGSERIAL PRIMARY KEY,
  exam_id BIGINT NOT NULL REFERENCES exams(id) ON DELETE CASCADE,
  user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  status VARCHAR(20) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'inactive')),
  assigned_by BIGINT REFERENCES users(id),
  assigned_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (exam_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_exam_assignments_exam_status
  ON exam_assignments (exam_id, status, user_id);

CREATE INDEX IF NOT EXISTS idx_exam_assignments_user_status
  ON exam_assignments (user_id, status, exam_id);
