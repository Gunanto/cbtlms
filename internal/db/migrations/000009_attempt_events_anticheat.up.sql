BEGIN;

CREATE TABLE IF NOT EXISTS attempt_events (
  id BIGSERIAL PRIMARY KEY,
  attempt_id BIGINT NOT NULL REFERENCES attempts(id) ON DELETE CASCADE,
  event_type VARCHAR(30) NOT NULL CHECK (event_type IN ('tab_blur','reconnect','rapid_refresh','fullscreen_exit')),
  payload JSONB NOT NULL DEFAULT '{}'::jsonb,
  client_ts TIMESTAMPTZ,
  server_ts TIMESTAMPTZ NOT NULL DEFAULT now(),
  actor_user_id BIGINT REFERENCES users(id),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_attempt_events_attempt_server
  ON attempt_events (attempt_id, server_ts DESC);

CREATE INDEX IF NOT EXISTS idx_attempt_events_type_server
  ON attempt_events (event_type, server_ts DESC);

COMMIT;
