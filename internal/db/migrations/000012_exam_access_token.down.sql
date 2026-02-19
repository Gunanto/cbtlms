ALTER TABLE exams
  DROP COLUMN IF EXISTS exam_token_generated_by,
  DROP COLUMN IF EXISTS exam_token_generated_at,
  DROP COLUMN IF EXISTS exam_token_expires_at,
  DROP COLUMN IF EXISTS exam_token_hash;

DROP INDEX IF EXISTS idx_exams_token_active;
