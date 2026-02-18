BEGIN;

ALTER TABLE users
  ADD COLUMN IF NOT EXISTS email VARCHAR(255),
  ADD COLUMN IF NOT EXISTS email_verified_at TIMESTAMPTZ,
  ADD COLUMN IF NOT EXISTS account_status VARCHAR(30) NOT NULL DEFAULT 'active',
  ADD COLUMN IF NOT EXISTS approved_by BIGINT REFERENCES users(id),
  ADD COLUMN IF NOT EXISTS approved_at TIMESTAMPTZ;

ALTER TABLE users
  DROP CONSTRAINT IF EXISTS chk_users_account_status;

ALTER TABLE users
  ADD CONSTRAINT chk_users_account_status
  CHECK (account_status IN ('pending_verification','active','rejected','suspended'));

CREATE UNIQUE INDEX IF NOT EXISTS ux_users_email_not_null
  ON users (email)
  WHERE email IS NOT NULL;

CREATE TABLE IF NOT EXISTS auth_otp_codes (
  id BIGSERIAL PRIMARY KEY,
  user_id BIGINT REFERENCES users(id) ON DELETE CASCADE,
  email VARCHAR(255) NOT NULL,
  code_hash TEXT NOT NULL,
  purpose VARCHAR(30) NOT NULL CHECK (purpose IN ('login','verify_email','reset_password')),
  expires_at TIMESTAMPTZ NOT NULL,
  used_at TIMESTAMPTZ,
  attempt_count INT NOT NULL DEFAULT 0,
  max_attempt INT NOT NULL DEFAULT 5,
  resend_count INT NOT NULL DEFAULT 0,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CHECK (attempt_count >= 0),
  CHECK (max_attempt > 0),
  CHECK (resend_count >= 0)
);

CREATE TABLE IF NOT EXISTS registration_requests (
  id BIGSERIAL PRIMARY KEY,
  role_requested VARCHAR(20) NOT NULL CHECK (role_requested IN ('siswa','guru')),
  email VARCHAR(255) NOT NULL,
  password_hash TEXT NOT NULL,
  full_name VARCHAR(150) NOT NULL,
  phone VARCHAR(30),
  institution_name VARCHAR(200),
  form_payload JSONB NOT NULL DEFAULT '{}'::jsonb,
  status VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','approved','rejected')),
  review_note TEXT,
  reviewed_by BIGINT REFERENCES users(id),
  reviewed_at TIMESTAMPTZ,
  created_user_id BIGINT REFERENCES users(id),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS auth_identities (
  id BIGSERIAL PRIMARY KEY,
  user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  provider VARCHAR(30) NOT NULL CHECK (provider IN ('password','email_otp')),
  provider_key VARCHAR(255) NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (provider, provider_key),
  UNIQUE (user_id, provider)
);

CREATE INDEX IF NOT EXISTS idx_auth_otp_codes_email_purpose
  ON auth_otp_codes (email, purpose, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_auth_otp_codes_user_created
  ON auth_otp_codes (user_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_registration_requests_status_created
  ON registration_requests (status, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_registration_requests_email
  ON registration_requests (email);

CREATE INDEX IF NOT EXISTS idx_auth_identities_user
  ON auth_identities (user_id);

COMMIT;
