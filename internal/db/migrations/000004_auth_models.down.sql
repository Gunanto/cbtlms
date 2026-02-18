BEGIN;

DROP INDEX IF EXISTS idx_auth_identities_user;
DROP INDEX IF EXISTS idx_registration_requests_email;
DROP INDEX IF EXISTS idx_registration_requests_status_created;
DROP INDEX IF EXISTS idx_auth_otp_codes_user_created;
DROP INDEX IF EXISTS idx_auth_otp_codes_email_purpose;

DROP TABLE IF EXISTS auth_identities;
DROP TABLE IF EXISTS registration_requests;
DROP TABLE IF EXISTS auth_otp_codes;

DROP INDEX IF EXISTS ux_users_email_not_null;

ALTER TABLE users
  DROP CONSTRAINT IF EXISTS chk_users_account_status;

ALTER TABLE users
  DROP COLUMN IF EXISTS approved_at,
  DROP COLUMN IF EXISTS approved_by,
  DROP COLUMN IF EXISTS account_status,
  DROP COLUMN IF EXISTS email_verified_at,
  DROP COLUMN IF EXISTS email;

COMMIT;
