-- Trivial first migration that proves the goose chain works end to end.
-- Real domain tables (users, workspaces, channels, ...) arrive from Phase 2 on.

-- +goose Up
-- +goose StatementBegin
CREATE TABLE schema_smoke (
    id          BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    note        TEXT NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE schema_smoke;
-- +goose StatementEnd
