-- Per-channel publish permissions. A member can be restricted to a subset of the
-- workspace's connected channels for publishing. channel_restricted flips the
-- membership into allowlist mode; channel_grants lists the allowed channels.
-- Not restricted (the default) = full access to every channel (backward compatible).

-- +goose Up
-- +goose StatementBegin
ALTER TABLE workspace_members ADD COLUMN channel_restricted BOOLEAN NOT NULL DEFAULT false;
-- +goose StatementEnd
-- +goose StatementBegin
CREATE TABLE channel_grants (
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    channel_id   UUID NOT NULL REFERENCES channels(id) ON DELETE CASCADE,
    user_id      UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (channel_id, user_id)
);
-- +goose StatementEnd
-- +goose StatementBegin
CREATE INDEX idx_channel_grants_user ON channel_grants (workspace_id, user_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE channel_grants;
-- +goose StatementEnd
-- +goose StatementBegin
ALTER TABLE workspace_members DROP COLUMN channel_restricted;
-- +goose StatementEnd
