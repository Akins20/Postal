package worker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"

	"github.com/Akins20/postal/internal/publish"
)

// refreshLookahead is how far before expiry the periodic job refreshes tokens.
const refreshLookahead = time.Hour

// refreshBatch bounds channels refreshed per periodic run.
const refreshBatch = 100

// Scheduler is the schedule-domain surface the processor needs: load a job's
// publish context and record status transitions. schedule.Service satisfies it.
type Scheduler interface {
	Claim(ctx context.Context, jobID uuid.UUID) (bool, error)
	ExecutionContext(ctx context.Context, jobID uuid.UUID) (uuid.UUID, publish.PostVariant, error)
	MarkPublished(ctx context.Context, jobID uuid.UUID) error
	MarkFailed(ctx context.Context, jobID uuid.UUID, cause string) error
	MarkRetry(ctx context.Context, jobID uuid.UUID, cause string) error
}

// Publisher publishes a variant to a channel. publish.Pipeline satisfies it.
type Publisher interface {
	Publish(ctx context.Context, channelID uuid.UUID, v publish.PostVariant) (*publish.Result, error)
}

// TokenRefresher lists and refreshes channels nearing token expiry.
// channel.Service satisfies it.
type TokenRefresher interface {
	DueForRefresh(ctx context.Context, before time.Time, limit int32) ([]uuid.UUID, error)
	RefreshChannel(ctx context.Context, channelID uuid.UUID) error
}

// Processor handles asynq tasks.
type Processor struct {
	sched    Scheduler
	pipeline Publisher
	channels TokenRefresher
	log      *slog.Logger
	clock    func() time.Time
}

// NewProcessor builds a Processor. clock defaults to time.Now.
func NewProcessor(sched Scheduler, pipeline Publisher, channels TokenRefresher, log *slog.Logger, clock func() time.Time) *Processor {
	if clock == nil {
		clock = time.Now
	}
	return &Processor{sched: sched, pipeline: pipeline, channels: channels, log: log, clock: clock}
}

// ProcessPublish executes one scheduled publish job. Terminal failures are not
// retried (wrapped asynq.SkipRetry); retryable failures return an error so asynq
// retries. Idempotency (job ID key) ensures a retried task never double-posts.
func (p *Processor) ProcessPublish(ctx context.Context, t *asynq.Task) error {
	var pl publishPayload
	if err := json.Unmarshal(t.Payload(), &pl); err != nil {
		return fmt.Errorf("decoding payload: %v: %w", err, asynq.SkipRetry)
	}

	// Claim the job (scheduled -> publishing). If it can't be claimed it was
	// canceled or already handled — do NOT publish it.
	claimed, err := p.sched.Claim(ctx, pl.JobID)
	if err != nil {
		return fmt.Errorf("claiming job %s: %w", pl.JobID, err) // transient -> asynq retries
	}
	if !claimed {
		return fmt.Errorf("job %s not claimable (canceled or already handled): %w", pl.JobID, asynq.SkipRetry)
	}

	channelID, variant, err := p.sched.ExecutionContext(ctx, pl.JobID)
	if err != nil {
		// Variant gone (e.g. post deleted) after claim — fail the job, don't retry.
		_ = p.sched.MarkFailed(ctx, pl.JobID, err.Error())
		return fmt.Errorf("loading job %s: %v: %w", pl.JobID, err, asynq.SkipRetry)
	}

	_, err = p.pipeline.Publish(ctx, channelID, variant)
	if err == nil {
		return p.sched.MarkPublished(ctx, pl.JobID)
	}

	var ae *publish.Error
	if errors.As(err, &ae) && ae.Class == publish.ClassTerminal {
		_ = p.sched.MarkFailed(ctx, pl.JobID, ae.Error())
		return fmt.Errorf("terminal publish failure for job %s: %v: %w", pl.JobID, err, asynq.SkipRetry)
	}
	// Retryable (or exhausted): record and let asynq retry.
	_ = p.sched.MarkRetry(ctx, pl.JobID, err.Error())
	return fmt.Errorf("retryable publish failure for job %s: %w", pl.JobID, err)
}

// ProcessRefreshTokens refreshes channels whose credentials are near expiry.
// Per-channel failures are logged, not fatal, so one bad channel doesn't abort
// the batch.
func (p *Processor) ProcessRefreshTokens(ctx context.Context, _ *asynq.Task) error {
	ids, err := p.channels.DueForRefresh(ctx, p.clock().Add(refreshLookahead), refreshBatch)
	if err != nil {
		return fmt.Errorf("listing channels due for refresh: %w", err)
	}
	for _, id := range ids {
		if rErr := p.channels.RefreshChannel(ctx, id); rErr != nil && p.log != nil {
			p.log.WarnContext(ctx, "channel token refresh failed", slog.String("channel_id", id.String()), slog.String("error", rErr.Error()))
		}
	}
	return nil
}
