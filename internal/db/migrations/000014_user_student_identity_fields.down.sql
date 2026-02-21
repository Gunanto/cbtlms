DROP INDEX IF EXISTS ux_users_nisn_not_null;
DROP INDEX IF EXISTS ux_users_participant_no_not_null;

ALTER TABLE users
  DROP COLUMN IF EXISTS nisn,
  DROP COLUMN IF EXISTS participant_no;
