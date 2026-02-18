BEGIN;

ALTER TABLE exams
  DROP CONSTRAINT IF EXISTS chk_exams_review_policy;

COMMIT;
