-- Phase 5: posts and per-channel content variants (the composer). A post is the
-- logical unit; each post_variant is the content tailored for one channel
-- (enabling compose-once, multi-channel publishing). Scheduling is Phase 6.

-- +goose Up
-- +goose StatementBegin
CREATE TABLE posts (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id   UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    author_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    status         TEXT NOT NULL DEFAULT 'draft',
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX idx_posts_workspace_created ON posts (workspace_id, created_at DESC);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TABLE post_variants (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    post_id          UUID NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
    channel_id       UUID NOT NULL REFERENCES channels(id) ON DELETE CASCADE,
    body             TEXT NOT NULL DEFAULT '',
    media_refs       JSONB NOT NULL DEFAULT '[]'::jsonb,
    platform_options JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (post_id, channel_id)
);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX idx_post_variants_post ON post_variants (post_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE post_variants;
-- +goose StatementEnd
-- +goose StatementBegin
DROP TABLE posts;
-- +goose StatementEnd
