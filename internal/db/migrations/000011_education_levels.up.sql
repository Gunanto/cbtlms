CREATE TABLE IF NOT EXISTS education_levels (
  id BIGSERIAL PRIMARY KEY,
  name VARCHAR(60) NOT NULL UNIQUE,
  is_active BOOLEAN NOT NULL DEFAULT TRUE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

INSERT INTO education_levels (name, is_active, created_at, updated_at)
SELECT DISTINCT TRIM(education_level), TRUE, now(), now()
FROM subjects
WHERE TRIM(COALESCE(education_level, '')) <> ''
ON CONFLICT (name) DO NOTHING;
