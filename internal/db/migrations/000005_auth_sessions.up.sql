BEGIN;

CREATE TABLE IF NOT EXISTS auth_sessions (
  id BIGSERIAL PRIMARY KEY,
  user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  session_token_hash TEXT NOT NULL UNIQUE,
  expires_at TIMESTAMPTZ NOT NULL,
  revoked_at TIMESTAMPTZ,
  ip_address VARCHAR(64),
  user_agent TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_auth_sessions_user_expires
  ON auth_sessions (user_id, expires_at DESC);

CREATE INDEX IF NOT EXISTS idx_auth_sessions_active
  ON auth_sessions (expires_at)
  WHERE revoked_at IS NULL;

COMMIT;
