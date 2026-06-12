-- Phase 13.5: third-party workspace integrations (first: OGShortener link
-- shortening). Credentials are envelope-encrypted with the master key, same
-- as the channel token vault.

-- +goose Up
-- +goose StatementBegin
CREATE TABLE workspace_integrations (
    workspace_id  UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    provider      TEXT NOT NULL,            -- e.g. ogshortener
    enabled       BOOLEAN NOT NULL DEFAULT false,
    auto_apply    BOOLEAN NOT NULL DEFAULT false, -- reserved: apply at publish time
    credentials   BYTEA,                    -- AES-256-GCM sealed API key; NULL = not configured
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (workspace_id, provider)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE workspace_integrations;
-- +goose StatementEnd
