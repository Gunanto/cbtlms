BEGIN;

UPDATE exams
SET review_policy = 'after_submit'
WHERE review_policy IS NULL
   OR review_policy NOT IN ('after_submit','after_exam_end','immediate','disabled');

ALTER TABLE exams
  DROP CONSTRAINT IF EXISTS chk_exams_review_policy;

ALTER TABLE exams
  ADD CONSTRAINT chk_exams_review_policy
  CHECK (review_policy IN ('after_submit','after_exam_end','immediate','disabled'));

COMMIT;
