-- The audit log records sensitive actions (auth, channel connect/disconnect,
-- publish, deletes, role/capability changes) for security forensics. It is
-- append-only. Foreign keys to workspace/user are intentionally omitted until
-- those tables exist (Phase 2); the columns are nullable to allow system events
-- and pre-account actions (e.g. failed logins).

-- +goose Up
-- +goose StatementBegin
CREATE TABLE audit_log (
    id              BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    workspace_id    UUID,
    actor_user_id   UUID,
    action          TEXT NOT NULL,
    target          TEXT NOT NULL DEFAULT '',
    metadata        JSONB NOT NULL DEFAULT '{}'::jsonb,
    ip              TEXT NOT NULL DEFAULT '',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Common access patterns: by workspace over time, and by actor over time.
CREATE INDEX idx_audit_log_workspace_created ON audit_log (workspace_id, created_at DESC);
CREATE INDEX idx_audit_log_actor_created ON audit_log (actor_user_id, created_at DESC);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE audit_log;
-- +goose StatementEnd
