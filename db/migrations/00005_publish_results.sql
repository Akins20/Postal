-- Phase 4: records of successful publishes, keyed by an idempotency key so the
-- pipeline never double-posts on retry/re-run. post_id is nullable until the
-- posts domain lands (Phase 5).

-- +goose Up
-- +goose StatementBegin
CREATE TABLE publish_results (
    id               BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    channel_id       UUID NOT NULL REFERENCES channels(id) ON DELETE CASCADE,
    post_id          UUID,
    idempotency_key  TEXT NOT NULL UNIQUE,
    platform_post_id TEXT NOT NULL,
    raw_response     JSONB NOT NULL DEFAULT '{}'::jsonb,
    published_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX idx_publish_results_channel ON publish_results (channel_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE publish_results;
-- +goose StatementEnd
