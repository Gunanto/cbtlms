BEGIN;

CREATE TABLE IF NOT EXISTS auth_guard_states (
  id BIGSERIAL PRIMARY KEY,
  purpose VARCHAR(30) NOT NULL CHECK (purpose IN ('password_login','otp_verify','otp_request')),
  subject_key VARCHAR(255) NOT NULL,
  failed_count INT NOT NULL DEFAULT 0,
  locked_until TIMESTAMPTZ,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (purpose, subject_key),
  CHECK (failed_count >= 0)
);

CREATE INDEX IF NOT EXISTS idx_auth_guard_states_purpose_lock
  ON auth_guard_states (purpose, locked_until);

INSERT INTO users (
  username,
  password_hash,
  full_name,
  role,
  is_active,
  email,
  email_verified_at,
  account_status,
  approved_at,
  created_at,
  updated_at
)
VALUES
  (
    'admin',
    '$2b$12$2DH9TaJCXIxH0mwYwzJJ5OAkIFTV.xRpqXCKzxc1zFK9HC0hIpy.O',
    'Administrator CBT',
    'admin',
    TRUE,
    'admin@cbtlms.local',
    now(),
    'active',
    now(),
    now(),
    now()
  ),
  (
    'proktor',
    '$2b$12$LEniT5NtMYm4tv3TlSOCS.UCGEB9bJC9X5wRZQRFYsxLYfn3eaioW',
    'Proktor CBT',
    'proktor',
    TRUE,
    'proktor@cbtlms.local',
    now(),
    'active',
    now(),
    now(),
    now()
  )
ON CONFLICT (username)
DO NOTHING;

INSERT INTO auth_identities (user_id, provider, provider_key, created_at)
SELECT u.id, 'password', u.username, now()
FROM users u
WHERE u.username IN ('admin', 'proktor')
ON CONFLICT (provider, provider_key)
DO NOTHING;

COMMIT;
