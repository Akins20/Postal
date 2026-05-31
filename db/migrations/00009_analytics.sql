-- Phase 8: post-performance analytics. A poller fetches public metrics for
-- published posts from each platform and appends a time-series snapshot. Long
-- format (one row per metric per capture) keeps it platform-agnostic. A
-- per-(channel, platform post) poll-state row tracks cadence and terminal stop
-- so a deleted/failed post isn't re-polled every sweep.

-- +goose Up
-- +goose StatementBegin
CREATE TABLE metric_snapshots (
    id               BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    workspace_id     UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    channel_id       UUID NOT NULL REFERENCES channels(id) ON DELETE CASCADE,
    post_id          UUID,            -- null for posts published before the composer (Phase 5)
    platform_post_id TEXT NOT NULL,   -- the platform's post identifier
    metric           TEXT NOT NULL,   -- likes | reposts | replies | impressions | ...
    value            BIGINT NOT NULL,
    captured_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
-- +goose StatementEnd

-- Workspace reporting groups by (post, channel) — a compose-once post fans out
-- to multiple channels, each with its own metrics — so the latest-per-metric
-- DISTINCT ON keys on (post_id, channel_id, metric).
-- +goose StatementBegin
CREATE INDEX idx_metric_snapshots_ws_post ON metric_snapshots (workspace_id, post_id, channel_id, metric, captured_at DESC);
-- +goose StatementEnd

-- metric_poll_state tracks, per (channel, platform post), the last poll time and
-- whether polling is done (the post was deleted at the platform), so the poller
-- dedups by attempt rather than by snapshot existence.
-- +goose StatementBegin
CREATE TABLE metric_poll_state (
    channel_id       UUID NOT NULL REFERENCES channels(id) ON DELETE CASCADE,
    platform_post_id TEXT NOT NULL,
    last_polled_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    done             BOOLEAN NOT NULL DEFAULT FALSE,
    PRIMARY KEY (channel_id, platform_post_id)
);
-- +goose StatementEnd

-- The poll's driving query filters publish_results by published_at; index it so
-- the 7-day-window scan stays a range scan as publish history grows.
-- +goose StatementBegin
CREATE INDEX idx_publish_results_published ON publish_results (published_at DESC);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX idx_publish_results_published;
-- +goose StatementEnd
-- +goose StatementBegin
DROP TABLE metric_poll_state;
-- +goose StatementEnd
-- +goose StatementBegin
DROP TABLE metric_snapshots;
-- +goose StatementEnd
