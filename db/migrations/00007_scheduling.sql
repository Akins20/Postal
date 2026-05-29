-- Phase 6: the scheduling engine. schedule_slots define a channel's recurring
-- posting times (queue-based scheduling); scheduled_jobs are concrete publish
-- jobs at a UTC run_at, executed by the asynq worker via the publish pipeline.

-- +goose Up
-- +goose StatementBegin
CREATE TABLE schedule_slots (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    channel_id  UUID NOT NULL REFERENCES channels(id) ON DELETE CASCADE,
    day_of_week SMALLINT NOT NULL,   -- 0=Sunday .. 6=Saturday
    time_of_day TEXT NOT NULL,        -- "HH:MM" in the slot's timezone
    timezone    TEXT NOT NULL,        -- IANA tz, e.g. "America/New_York"
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (day_of_week BETWEEN 0 AND 6)
);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX idx_schedule_slots_channel ON schedule_slots (channel_id);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TABLE scheduled_jobs (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    post_id       UUID NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
    channel_id    UUID NOT NULL REFERENCES channels(id) ON DELETE CASCADE,
    run_at        TIMESTAMPTZ NOT NULL,
    status        TEXT NOT NULL DEFAULT 'scheduled', -- scheduled|publishing|published|failed|canceled
    attempts      INTEGER NOT NULL DEFAULT 0,
    last_error    TEXT NOT NULL DEFAULT '',
    asynq_task_id TEXT NOT NULL DEFAULT '',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX idx_scheduled_jobs_post ON scheduled_jobs (post_id);
-- +goose StatementEnd

-- +goose StatementBegin
-- Calendar range queries: by channel over a time window.
CREATE INDEX idx_scheduled_jobs_channel_runat ON scheduled_jobs (channel_id, run_at);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE scheduled_jobs;
-- +goose StatementEnd
-- +goose StatementBegin
DROP TABLE schedule_slots;
-- +goose StatementEnd
