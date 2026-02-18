BEGIN;

DROP INDEX IF EXISTS idx_auth_sessions_active;
DROP INDEX IF EXISTS idx_auth_sessions_user_expires;
DROP TABLE IF EXISTS auth_sessions;

COMMIT;
