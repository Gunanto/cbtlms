BEGIN;

CREATE OR REPLACE FUNCTION set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
  NEW.updated_at = now();
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_users_set_updated_at ON users;
CREATE TRIGGER trg_users_set_updated_at
BEFORE UPDATE ON users
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

DROP TRIGGER IF EXISTS trg_questions_set_updated_at ON questions;
CREATE TRIGGER trg_questions_set_updated_at
BEFORE UPDATE ON questions
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

DROP TRIGGER IF EXISTS trg_stimuli_set_updated_at ON stimuli;
CREATE TRIGGER trg_stimuli_set_updated_at
BEFORE UPDATE ON stimuli
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

DO $$
DECLARE
  v_writer_indicator_id BIGINT;
  v_writer_question_id BIGINT;
  v_reviewer_id BIGINT;
  v_subject_id BIGINT;
  v_question_id BIGINT;
  v_exam_id BIGINT;
  v_stimulus_id BIGINT;
  v_qv2_id BIGINT;
  v_qv3_id BIGINT;
BEGIN
  INSERT INTO users (username, password_hash, full_name, role, is_active)
  VALUES
    ('demo_penulis_indikator', 'demo_hash', 'DEMO Penulis Indikator', 'guru', TRUE),
    ('demo_penulis_soal', 'demo_hash', 'DEMO Penulis Soal', 'guru', TRUE),
    ('demo_penelaah_soal', 'demo_hash', 'DEMO Penelaah Soal', 'guru', TRUE)
  ON CONFLICT (username) DO NOTHING;

  SELECT id INTO v_writer_indicator_id FROM users WHERE username = 'demo_penulis_indikator';
  SELECT id INTO v_writer_question_id FROM users WHERE username = 'demo_penulis_soal';
  SELECT id INTO v_reviewer_id FROM users WHERE username = 'demo_penelaah_soal';

  SELECT id INTO v_subject_id
  FROM subjects
  WHERE education_level = 'SMA/MA/SMK/MAK'
    AND subject_type = 'Mata Pelajaran Wajib'
    AND name = 'Matematika DEMO';

  IF v_subject_id IS NULL THEN
    INSERT INTO subjects (education_level, subject_type, name, is_active)
    VALUES ('SMA/MA/SMK/MAK', 'Mata Pelajaran Wajib', 'Matematika DEMO', TRUE)
    RETURNING id INTO v_subject_id;
  END IF;

  SELECT id INTO v_question_id
  FROM questions
  WHERE subject_id = v_subject_id
    AND stem_html = '<p>DEMO: Nilai dari 2 + 3 adalah ...</p>'
  LIMIT 1;

  IF v_question_id IS NULL THEN
    INSERT INTO questions (
      subject_id,
      question_type,
      stem_html,
      stimulus_html,
      difficulty,
      metadata,
      is_active,
      version
    ) VALUES (
      v_subject_id,
      'pg_tunggal',
      '<p>DEMO: Nilai dari 2 + 3 adalah ...</p>',
      NULL,
      1,
      '{"source":"DEMO"}'::jsonb,
      TRUE,
      1
    )
    RETURNING id INTO v_question_id;

    INSERT INTO question_options (question_id, option_key, option_html, is_correct)
    VALUES
      (v_question_id, 'A', '<p>4</p>', FALSE),
      (v_question_id, 'B', '<p>5</p>', TRUE),
      (v_question_id, 'C', '<p>6</p>', FALSE),
      (v_question_id, 'D', '<p>7</p>', FALSE);
  END IF;

  INSERT INTO exams (
    code,
    title,
    subject_id,
    duration_minutes,
    randomize_questions,
    randomize_options,
    review_policy,
    is_active,
    created_by
  ) VALUES (
    'DEMO-TKA-MTK-001',
    'DEMO Simulasi TKA Matematika',
    v_subject_id,
    90,
    FALSE,
    FALSE,
    'after_submit',
    TRUE,
    v_writer_indicator_id
  )
  ON CONFLICT (code) DO NOTHING;

  SELECT id INTO v_exam_id FROM exams WHERE code = 'DEMO-TKA-MTK-001';

  INSERT INTO exam_questions (exam_id, question_id, seq_no, weight)
  VALUES (v_exam_id, v_question_id, 1, 1.0)
  ON CONFLICT (exam_id, question_id) DO NOTHING;

  SELECT id INTO v_stimulus_id
  FROM stimuli
  WHERE subject_id = v_subject_id
    AND title = 'DEMO Stimulus Operasi Bilangan'
  LIMIT 1;

  IF v_stimulus_id IS NULL THEN
    INSERT INTO stimuli (subject_id, title, stimulus_type, content, is_active, created_by)
    VALUES (
      v_subject_id,
      'DEMO Stimulus Operasi Bilangan',
      'single',
      '{"tabs":[{"title":"Stimulus 1","html":"<p>DEMO: Gunakan operasi penjumlahan dasar.</p>"}]}'::jsonb,
      TRUE,
      v_writer_question_id
    )
    RETURNING id INTO v_stimulus_id;
  END IF;

  INSERT INTO question_versions (
    question_id,
    version_no,
    stimulus_id,
    stem_html,
    explanation_html,
    hint_html,
    answer_key,
    status,
    is_public,
    is_active,
    duration_seconds,
    weight,
    change_note,
    created_by
  ) VALUES
    (
      v_question_id,
      1,
      v_stimulus_id,
      '<p>DEMO v1: Nilai dari 2 + 3 adalah ...</p>',
      '<p>DEMO: 2 + 3 = 5</p>',
      '<p>DEMO: Jumlahkan dua bilangan.</p>',
      '{"selected":["B"]}'::jsonb,
      'draft',
      FALSE,
      TRUE,
      30,
      1.0,
      'DEMO: draft awal oleh penulis soal',
      v_writer_question_id
    ),
    (
      v_question_id,
      2,
      v_stimulus_id,
      '<p>DEMO v2: Nilai dari 2 + 3 adalah ...</p>',
      '<p>DEMO: 2 + 3 = 5</p>',
      '<p>DEMO: Fokus pada operasi dasar.</p>',
      '{"selected":["B"]}'::jsonb,
      'menunggu_reviu',
      FALSE,
      TRUE,
      30,
      1.0,
      'DEMO: diajukan untuk reviu',
      v_writer_question_id
    ),
    (
      v_question_id,
      3,
      v_stimulus_id,
      '<p>DEMO v3: Berapakah hasil dari 2 + 3?</p>',
      '<p>DEMO: Hasil penjumlahan 2 dan 3 adalah 5.</p>',
      '<p>DEMO: Hitung bertahap.</p>',
      '{"selected":["B"]}'::jsonb,
      'final',
      TRUE,
      TRUE,
      30,
      1.0,
      'DEMO: final setelah revisi',
      v_writer_question_id
    )
  ON CONFLICT (question_id, version_no) DO NOTHING;

  SELECT id INTO v_qv2_id FROM question_versions WHERE question_id = v_question_id AND version_no = 2;
  SELECT id INTO v_qv3_id FROM question_versions WHERE question_id = v_question_id AND version_no = 3;

  INSERT INTO review_tasks (question_version_id, exam_id, reviewer_id, status, assigned_at, reviewed_at, note)
  VALUES
    (
      v_qv2_id,
      v_exam_id,
      v_reviewer_id,
      'perlu_revisi',
      now(),
      now(),
      'DEMO: perlu perbaikan redaksi soal'
    ),
    (
      v_qv3_id,
      v_exam_id,
      v_reviewer_id,
      'disetujui',
      now(),
      now(),
      'DEMO: soal disetujui untuk digunakan'
    )
  ON CONFLICT (question_version_id, reviewer_id) DO NOTHING;

  INSERT INTO review_comments (review_task_id, author_id, comment_type, comment_text)
  SELECT rt.id, v_reviewer_id, 'revisi', 'DEMO: Ubah kalimat pokok soal agar lebih jelas.'
  FROM review_tasks rt
  WHERE rt.question_version_id = v_qv2_id
    AND NOT EXISTS (
      SELECT 1 FROM review_comments rc
      WHERE rc.review_task_id = rt.id
        AND rc.comment_text = 'DEMO: Ubah kalimat pokok soal agar lebih jelas.'
    );

  INSERT INTO review_comments (review_task_id, author_id, comment_type, comment_text)
  SELECT rt.id, v_reviewer_id, 'keputusan', 'DEMO: Revisi diterima, soal disetujui.'
  FROM review_tasks rt
  WHERE rt.question_version_id = v_qv3_id
    AND NOT EXISTS (
      SELECT 1 FROM review_comments rc
      WHERE rc.review_task_id = rt.id
        AND rc.comment_text = 'DEMO: Revisi diterima, soal disetujui.'
    );

  INSERT INTO question_parallels (exam_id, question_id, parallel_group, parallel_order, parallel_label, is_active)
  VALUES
    (v_exam_id, v_question_id, 'demo_flow', 1, 'pararel_1', TRUE)
  ON CONFLICT (exam_id, parallel_group, parallel_order) DO NOTHING;
END $$;

COMMIT;
