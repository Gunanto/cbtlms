BEGIN;

DELETE FROM auth_identities
WHERE provider = 'password'
  AND provider_key IN ('admin', 'proktor');

DELETE FROM auth_sessions
WHERE user_id IN (
  SELECT id FROM users WHERE username IN ('admin', 'proktor')
);

DELETE FROM users
WHERE username IN ('admin', 'proktor');

DROP INDEX IF EXISTS idx_auth_guard_states_purpose_lock;
DROP TABLE IF EXISTS auth_guard_states;

COMMIT;
