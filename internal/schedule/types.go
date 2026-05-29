// Package schedule implements the scheduling engine: per-channel posting slots
// (queue-based scheduling), concrete scheduled jobs at a UTC run_at, calendar
// queries, and cancel. Jobs are enqueued to asynq (via the Enqueuer) and
// executed by the worker (internal/worker) through the publish pipeline.
package schedule

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Job statuses.
const (
	StatusScheduled  = "scheduled"
	StatusPublishing = "publishing"
	StatusPublished  = "published"
	StatusFailed     = "failed"
	StatusCanceled   = "canceled"
)

// Job is a concrete publish job for one post variant (channel) at run_at.
type Job struct {
	ID        uuid.UUID `json:"id"`
	PostID    uuid.UUID `json:"post_id"`
	ChannelID uuid.UUID `json:"channel_id"`
	RunAt     time.Time `json:"run_at"`
	Status    string    `json:"status"`
	Attempts  int       `json:"attempts"`
	LastError string    `json:"last_error,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// Slot is a recurring posting time for a channel (queue-based scheduling).
type Slot struct {
	ID        uuid.UUID `json:"id"`
	ChannelID uuid.UUID `json:"channel_id"`
	DayOfWeek int       `json:"day_of_week"` // 0=Sunday .. 6=Saturday
	TimeOfDay string    `json:"time_of_day"` // "HH:MM"
	Timezone  string    `json:"timezone"`    // IANA
	CreatedAt time.Time `json:"created_at"`
}

// Enqueuer schedules and cancels asynq publish tasks. The worker's asynq client
// satisfies it; defined here so the schedule domain does not import the worker.
type Enqueuer interface {
	// EnqueuePublish schedules a publish task for jobID to run at runAt, returning
	// the asynq task ID.
	EnqueuePublish(ctx context.Context, jobID uuid.UUID, runAt time.Time) (string, error)
	// Cancel removes a not-yet-run task by its asynq task ID (best-effort).
	Cancel(ctx context.Context, taskID string) error
}
