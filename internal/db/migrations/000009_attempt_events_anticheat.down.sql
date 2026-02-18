BEGIN;

DROP INDEX IF EXISTS idx_attempt_events_type_server;
DROP INDEX IF EXISTS idx_attempt_events_attempt_server;
DROP TABLE IF EXISTS attempt_events;

COMMIT;
