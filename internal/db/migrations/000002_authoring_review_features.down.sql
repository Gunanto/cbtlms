BEGIN;

DROP INDEX IF EXISTS idx_rc_review_task;
DROP INDEX IF EXISTS idx_rt_reviewer_status;
DROP INDEX IF EXISTS idx_qp_exam_group;
DROP INDEX IF EXISTS idx_qv_question_status;
DROP INDEX IF EXISTS idx_stimuli_subject;

DROP TABLE IF EXISTS review_comments;
DROP TABLE IF EXISTS review_tasks;
DROP TABLE IF EXISTS question_parallels;
DROP TABLE IF EXISTS question_versions;
DROP TABLE IF EXISTS stimuli;

COMMIT;
