CREATE TABLE IF NOT EXISTS question_reopen_requests (
  id BIGSERIAL PRIMARY KEY,
  question_id BIGINT NOT NULL REFERENCES questions(id) ON DELETE CASCADE,
  version_no INT NOT NULL,
  reason TEXT NOT NULL,
  status VARCHAR(20) NOT NULL CHECK (status IN ('pending','approved','rejected','cancelled')),
  requested_by BIGINT NOT NULL REFERENCES users(id),
  requested_role VARCHAR(20) NOT NULL CHECK (requested_role IN ('admin','proktor','guru')),
  requested_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  decided_by BIGINT REFERENCES users(id),
  decided_role VARCHAR(20) CHECK (decided_role IN ('admin','proktor','guru')),
  decided_note TEXT,
  decided_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_question_reopen_requests_lookup
  ON question_reopen_requests (question_id, version_no, status, requested_at DESC);

CREATE TABLE IF NOT EXISTS question_reopen_approvals (
  id BIGSERIAL PRIMARY KEY,
  reopen_request_id BIGINT NOT NULL REFERENCES question_reopen_requests(id) ON DELETE CASCADE,
  approver_id BIGINT NOT NULL REFERENCES users(id),
  approver_role VARCHAR(20) NOT NULL CHECK (approver_role IN ('admin','proktor','guru')),
  note TEXT,
  approved_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (reopen_request_id, approver_id),
  UNIQUE (reopen_request_id, approver_role)
);

CREATE INDEX IF NOT EXISTS idx_question_reopen_approvals_req
  ON question_reopen_approvals (reopen_request_id, approved_at DESC);
