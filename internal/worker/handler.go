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

	"github.com/Akins20/postal/internal/billing"
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

// PublishBiller charges a workspace's wallet for a paid-platform publish at
// claim time and refunds after a terminal failure. Free platforms are no-ops.
// billing.Service satisfies it; nil disables billing (everything free).
type PublishBiller interface {
	ChargePublish(ctx context.Context, jobID, channelID uuid.UUID) error
	RefundPublish(ctx context.Context, jobID, channelID uuid.UUID) error
}

// TokenRefresher lists and refreshes channels nearing token expiry.
// channel.Service satisfies it.
type TokenRefresher interface {
	DueForRefresh(ctx context.Context, before time.Time, limit int32) ([]uuid.UUID, error)
	RefreshChannel(ctx context.Context, channelID uuid.UUID) error
}

// AnalyticsPoller fetches and stores fresh metric snapshots for due posts,
// returning how many were polled and how many failed. analytics.Service
// satisfies it; nil disables the metrics sweep.
type AnalyticsPoller interface {
	PollMetrics(ctx context.Context) (polled, failed int, err error)
}

// Processor handles asynq tasks.
type Processor struct {
	sched     Scheduler
	pipeline  Publisher
	channels  TokenRefresher
	analytics AnalyticsPoller
	biller    PublishBiller
	log       *slog.Logger
	clock     func() time.Time
}

// NewProcessor builds a Processor. analytics may be nil (metrics sweep disabled);
// biller may be nil (billing disabled); clock defaults to time.Now.
func NewProcessor(sched Scheduler, pipeline Publisher, channels TokenRefresher, analytics AnalyticsPoller, biller PublishBiller, log *slog.Logger, clock func() time.Time) *Processor {
	if clock == nil {
		clock = time.Now
	}
	return &Processor{sched: sched, pipeline: pipeline, channels: channels, analytics: analytics, biller: biller, log: log, clock: clock}
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
		// A retryable load failure (e.g. a transient storage outage while
		// fetching attached media) must not permanently fail the job.
		var ae *publish.Error
		if errors.As(err, &ae) && ae.Class == publish.ClassRetryable {
			_ = p.sched.MarkRetry(ctx, pl.JobID, ae.Error())
			return fmt.Errorf("loading job %s (retryable): %w", pl.JobID, err)
		}
		// Terminal (variant/post gone, media deleted, malformed refs) — fail, no retry.
		_ = p.sched.MarkFailed(ctx, pl.JobID, err.Error())
		return fmt.Errorf("loading job %s: %v: %w", pl.JobID, err, asynq.SkipRetry)
	}

	if err := p.chargeForPublish(ctx, pl.JobID, channelID); err != nil {
		return err
	}

	_, err = p.pipeline.Publish(ctx, channelID, variant)
	if err == nil {
		return p.sched.MarkPublished(ctx, pl.JobID)
	}

	var ae *publish.Error
	if errors.As(err, &ae) && ae.Class == publish.ClassTerminal {
		// The publish definitively didn't happen — return the charge.
		if p.biller != nil {
			if rErr := p.biller.RefundPublish(ctx, pl.JobID, channelID); rErr != nil && p.log != nil {
				p.log.ErrorContext(ctx, "refund after terminal publish failure failed",
					slog.String("job_id", pl.JobID.String()), slog.String("error", rErr.Error()))
			}
		}
		_ = p.sched.MarkFailed(ctx, pl.JobID, ae.Error())
		return fmt.Errorf("terminal publish failure for job %s: %v: %w", pl.JobID, err, asynq.SkipRetry)
	}
	// Retryable (or exhausted): record and let asynq retry.
	_ = p.sched.MarkRetry(ctx, pl.JobID, err.Error())
	return fmt.Errorf("retryable publish failure for job %s: %w", pl.JobID, err)
}

// chargeForPublish deducts the wallet cost before publishing (X is pay-per-use;
// free platforms are a no-op). The charge is idempotent by job ID, so a
// re-claimed job never double-charges. No funds = permanent failure the user
// fixes by topping up.
func (p *Processor) chargeForPublish(ctx context.Context, jobID, channelID uuid.UUID) error {
	if p.biller == nil {
		return nil
	}
	err := p.biller.ChargePublish(ctx, jobID, channelID)
	switch {
	case err == nil:
		return nil
	case errors.Is(err, billing.ErrInsufficientCredits):
		_ = p.sched.MarkFailed(ctx, jobID, "insufficient wallet credits — top up to publish to this platform")
		return fmt.Errorf("insufficient credits for job %s: %w", jobID, asynq.SkipRetry)
	default:
		_ = p.sched.MarkRetry(ctx, jobID, err.Error())
		return fmt.Errorf("charging wallet for job %s: %w", jobID, err)
	}
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

// ProcessFetchMetrics polls analytics for recently-published posts. Per-post
// failures are logged inside the poller; a batch-level error is logged here but
// not retried (the next periodic run picks up where this left off).
func (p *Processor) ProcessFetchMetrics(ctx context.Context, _ *asynq.Task) error {
	if p.analytics == nil {
		return nil
	}
	polled, failed, err := p.analytics.PollMetrics(ctx)
	if err != nil {
		return fmt.Errorf("polling metrics: %w", err) // batch-level failure -> asynq retries
	}
	if failed > 0 && p.log != nil {
		p.log.WarnContext(ctx, "analytics poll skipped some posts", slog.Int("polled", polled), slog.Int("failed", failed))
	}
	return nil
}
