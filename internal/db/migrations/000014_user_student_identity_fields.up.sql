ALTER TABLE users
  ADD COLUMN IF NOT EXISTS participant_no VARCHAR(64),
  ADD COLUMN IF NOT EXISTS nisn VARCHAR(32);

CREATE UNIQUE INDEX IF NOT EXISTS ux_users_participant_no_not_null
  ON users (participant_no)
  WHERE participant_no IS NOT NULL AND btrim(participant_no) <> '';

CREATE UNIQUE INDEX IF NOT EXISTS ux_users_nisn_not_null
  ON users (nisn)
  WHERE nisn IS NOT NULL AND btrim(nisn) <> '';
