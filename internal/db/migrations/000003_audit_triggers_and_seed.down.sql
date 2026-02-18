BEGIN;

DELETE FROM review_comments
WHERE comment_text LIKE 'DEMO:%';

DELETE FROM review_tasks
WHERE note LIKE 'DEMO:%'
   OR question_version_id IN (
    SELECT qv.id
    FROM question_versions qv
    WHERE qv.change_note LIKE 'DEMO:%'
  );

DELETE FROM question_parallels
WHERE parallel_group = 'demo_flow';

DELETE FROM question_versions
WHERE change_note LIKE 'DEMO:%';

DELETE FROM stimuli
WHERE title = 'DEMO Stimulus Operasi Bilangan';

DELETE FROM exam_questions
WHERE exam_id IN (SELECT id FROM exams WHERE code = 'DEMO-TKA-MTK-001');

DELETE FROM exams
WHERE code = 'DEMO-TKA-MTK-001';

DELETE FROM question_options
WHERE question_id IN (
  SELECT id
  FROM questions
  WHERE stem_html = '<p>DEMO: Nilai dari 2 + 3 adalah ...</p>'
);

DELETE FROM questions
WHERE stem_html = '<p>DEMO: Nilai dari 2 + 3 adalah ...</p>';

DELETE FROM subjects
WHERE name = 'Matematika DEMO'
  AND education_level = 'SMA/MA/SMK/MAK'
  AND subject_type = 'Mata Pelajaran Wajib';

DELETE FROM users
WHERE username IN ('demo_penulis_indikator', 'demo_penulis_soal', 'demo_penelaah_soal');

DROP TRIGGER IF EXISTS trg_stimuli_set_updated_at ON stimuli;
DROP TRIGGER IF EXISTS trg_questions_set_updated_at ON questions;
DROP TRIGGER IF EXISTS trg_users_set_updated_at ON users;

DROP FUNCTION IF EXISTS set_updated_at();

COMMIT;
