-- Phase 3: connected social accounts (channels) and their encrypted OAuth
-- credentials. Credentials live in a separate, tightly-scoped table; tokens are
-- stored only as AES-256-GCM envelope-encrypted ciphertext (BYTEA), never plaintext.

-- +goose Up
-- +goose StatementBegin
CREATE TABLE channels (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id        UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    platform            TEXT NOT NULL,
    platform_account_id TEXT NOT NULL,
    handle              TEXT NOT NULL DEFAULT '',
    display_name        TEXT NOT NULL DEFAULT '',
    status              TEXT NOT NULL DEFAULT 'active',
    connected_by        UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (workspace_id, platform, platform_account_id)
);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX idx_channels_workspace ON channels (workspace_id);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TABLE channel_credentials (
    channel_id              UUID PRIMARY KEY REFERENCES channels(id) ON DELETE CASCADE,
    encrypted_access_token  BYTEA NOT NULL,
    encrypted_refresh_token BYTEA,
    scopes                  TEXT[] NOT NULL DEFAULT '{}',
    expires_at              TIMESTAMPTZ,
    key_version             INTEGER NOT NULL,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT now()
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE channel_credentials;
-- +goose StatementEnd
-- +goose StatementBegin
DROP TABLE channels;
-- +goose StatementEnd
