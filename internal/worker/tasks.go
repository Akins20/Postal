// Package worker runs the asynq job processor: it executes scheduled publish
// jobs through the publish pipeline and periodically refreshes channel tokens.
// It also exposes the asynq Client, which satisfies schedule.Enqueuer so the
// API server can enqueue publish tasks.
package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
)

// Task type names.
const (
	// TypePublish publishes one scheduled job (one post variant to one channel).
	TypePublish = "publish:variant"
	// TypeRefreshTokens refreshes channel credentials nearing expiry (periodic).
	TypeRefreshTokens = "channels:refresh_tokens"
	// TypeFetchMetrics polls analytics for recently-published posts (periodic).
	TypeFetchMetrics = "analytics:fetch_metrics"
)

// defaultQueue is the asynq queue publish/refresh tasks use.
const defaultQueue = "default"

// publishMaxRetry bounds asynq-level retries of a publish task (in addition to
// the pipeline's in-call retries).
const publishMaxRetry = 5

// publishPayload is the JSON payload of a publish task.
type publishPayload struct {
	JobID uuid.UUID `json:"job_id"`
}

// Client wraps the asynq client + inspector to enqueue and cancel tasks.
type Client struct {
	client    *asynq.Client
	inspector *asynq.Inspector
}

// NewClient builds an asynq Client over the given Redis options.
func NewClient(redis asynq.RedisClientOpt) *Client {
	return &Client{client: asynq.NewClient(redis), inspector: asynq.NewInspector(redis)}
}

// EnqueuePublish schedules a publish task for jobID at runAt, returning the
// asynq task ID (stored so the job can be canceled).
func (c *Client) EnqueuePublish(ctx context.Context, jobID uuid.UUID, runAt time.Time) (string, error) {
	payload, err := json.Marshal(publishPayload{JobID: jobID})
	if err != nil {
		return "", fmt.Errorf("encoding publish payload: %w", err)
	}
	info, err := c.client.EnqueueContext(ctx, asynq.NewTask(TypePublish, payload),
		asynq.ProcessAt(runAt), asynq.MaxRetry(publishMaxRetry), asynq.Queue(defaultQueue))
	if err != nil {
		return "", fmt.Errorf("enqueuing publish task: %w", err)
	}
	return info.ID, nil
}

// Cancel deletes a not-yet-run task by ID (best-effort; ignores not-found).
func (c *Client) Cancel(_ context.Context, taskID string) error {
	return c.inspector.DeleteTask(defaultQueue, taskID)
}

// Close releases the client/inspector resources.
func (c *Client) Close() error {
	_ = c.inspector.Close()
	return c.client.Close()
}
