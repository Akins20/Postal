-- Phase 7: uploaded media assets. The binary lives in object storage (MinIO/S3)
-- under storage_key; this table holds the metadata used for validation, quota,
-- and attaching media to post variants.

-- +goose Up
-- +goose StatementBegin
CREATE TABLE media_assets (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id  UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    kind          TEXT NOT NULL,            -- image | gif | video
    storage_key   TEXT NOT NULL,            -- object-storage key
    mime          TEXT NOT NULL,
    width         INTEGER NOT NULL DEFAULT 0,
    height        INTEGER NOT NULL DEFAULT 0,
    duration_ms   INTEGER NOT NULL DEFAULT 0, -- video duration; 0 if unknown
    bytes         BIGINT NOT NULL,
    status        TEXT NOT NULL DEFAULT 'ready', -- ready | processing | failed
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX idx_media_assets_workspace ON media_assets (workspace_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE media_assets;
-- +goose StatementEnd
