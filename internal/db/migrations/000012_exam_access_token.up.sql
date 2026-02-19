ALTER TABLE exams
  ADD COLUMN IF NOT EXISTS exam_token_hash TEXT,
  ADD COLUMN IF NOT EXISTS exam_token_expires_at TIMESTAMPTZ,
  ADD COLUMN IF NOT EXISTS exam_token_generated_at TIMESTAMPTZ,
  ADD COLUMN IF NOT EXISTS exam_token_generated_by BIGINT REFERENCES users(id);

CREATE INDEX IF NOT EXISTS idx_exams_token_active
  ON exams (id, exam_token_expires_at)
  WHERE exam_token_hash IS NOT NULL;
