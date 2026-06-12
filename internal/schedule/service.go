package schedule

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/Akins20/postal/internal/billing"
	"github.com/Akins20/postal/internal/platform/apperr"
	"github.com/Akins20/postal/internal/platform/db"
	"github.com/Akins20/postal/internal/platform/db/sqlc"
	"github.com/Akins20/postal/internal/publish"
	"github.com/Akins20/postal/internal/security"
)

// ChannelResolver verifies channel ownership within a workspace.
type ChannelResolver interface {
	PlatformFor(ctx context.Context, workspaceID, channelID uuid.UUID) (string, error)
}

// AffordabilityChecker verifies a workspace's wallet covers the paid-platform
// publishes about to be queued (the schedule-time soft gate; the worker holds
// the hard gate at claim time). billing.Service satisfies it; nil disables
// the gate (all platforms free).
type AffordabilityChecker interface {
	CheckAffordable(ctx context.Context, workspaceID uuid.UUID, items []billing.PublishItem) error
}

// maxPendingJobsPerWorkspace caps not-yet-completed scheduled jobs per workspace
// (anti-abuse: bounds queue depth and shared upstream-API load). Generous for
// real bulk scheduling; blocks runaway/scripted flooding. A var so tests can
// lower it without seeding thousands of rows.
var maxPendingJobsPerWorkspace int64 = 5000

// MediaLoader downloads an attached media asset's bytes for publishing, and
// presigns a public URL for platforms that fetch media themselves.
// media.Service satisfies it. Nil when the media pipeline is disabled.
type MediaLoader interface {
	OpenMedia(ctx context.Context, assetID uuid.UUID) (kind, mime string, data []byte, err error)
	MediaURL(ctx context.Context, assetID uuid.UUID, ttl time.Duration) (string, error)
}

// mediaURLTTL is how long presigned media links stay valid - generous enough
// for a platform's asynchronous container processing (IG allows up to 24h).
const mediaURLTTL = 2 * time.Hour

// Service implements scheduling over scheduled_jobs/schedule_slots, enqueuing
// publish tasks via the Enqueuer and exposing execution context to the worker.
type Service struct {
	pool     *db.Pool
	channels ChannelResolver
	enqueuer Enqueuer
	media    MediaLoader
	audit    security.Recorder
	biller   AffordabilityChecker
	clock    func() time.Time
}

// NewService builds a schedule Service. media may be nil (media disabled);
// biller may be nil (billing disabled); clock defaults to time.Now.
func NewService(pool *db.Pool, channels ChannelResolver, enqueuer Enqueuer, media MediaLoader, audit security.Recorder, biller AffordabilityChecker, clock func() time.Time) *Service {
	if clock == nil {
		clock = time.Now
	}
	return &Service{pool: pool, channels: channels, enqueuer: enqueuer, media: media, audit: audit, biller: biller, clock: clock}
}

// checkBilling runs the wallet soft gate for the platforms these variants
// target, mapping a shortfall to a user-actionable validation error.
func (s *Service) checkBilling(ctx context.Context, workspaceID uuid.UUID, variants []sqlc.PostVariant) error {
	if s.biller == nil {
		return nil
	}
	items := make([]billing.PublishItem, 0, len(variants))
	for _, v := range variants {
		platform, err := s.channels.PlatformFor(ctx, workspaceID, v.ChannelID)
		if err != nil {
			return err
		}
		media := string(v.MediaRefs)
		items = append(items, billing.PublishItem{
			Platform: platform,
			Body:     v.Body,
			HasMedia: media != "" && media != "null" && media != "[]",
		})
	}
	if err := s.biller.CheckAffordable(ctx, workspaceID, items); err != nil {
		if errors.Is(err, billing.ErrInsufficientCredits) {
			return apperr.Validation("insufficient_credits",
				"not enough wallet credits to schedule this. Top up on the Wallet page")
		}
		return err
	}
	return nil
}

// SchedulePost schedules every variant of a post to publish at runAt (UTC),
// creating a job per channel and enqueuing each. The post must belong to the
// workspace and have at least one variant.
func (s *Service) SchedulePost(ctx context.Context, workspaceID, postID uuid.UUID, runAt time.Time) ([]Job, error) {
	variants, err := s.postVariants(ctx, workspaceID, postID)
	if err != nil {
		return nil, err
	}
	if err := s.checkPendingQuota(ctx, workspaceID, len(variants)); err != nil {
		return nil, err
	}
	if err := s.checkBilling(ctx, workspaceID, variants); err != nil {
		return nil, err
	}
	jobs := make([]Job, 0, len(variants))
	for _, v := range variants {
		job, err := s.scheduleOne(ctx, postID, v.ChannelID, runAt.UTC())
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, job)
	}
	s.recordAudit(ctx, workspaceID, "post.schedule", postID.String())
	return jobs, nil
}

// ScheduleToSlots schedules each variant of a post into its channel's next open
// posting slot (queue-based scheduling).
func (s *Service) ScheduleToSlots(ctx context.Context, workspaceID, postID uuid.UUID) ([]Job, error) {
	variants, err := s.postVariants(ctx, workspaceID, postID)
	if err != nil {
		return nil, err
	}
	if err := s.checkPendingQuota(ctx, workspaceID, len(variants)); err != nil {
		return nil, err
	}
	if err := s.checkBilling(ctx, workspaceID, variants); err != nil {
		return nil, err
	}
	now := s.clock()
	jobs := make([]Job, 0, len(variants))
	for _, v := range variants {
		runAt, err := s.NextOpenSlot(ctx, v.ChannelID, now)
		if err != nil {
			return nil, err
		}
		job, err := s.scheduleOne(ctx, postID, v.ChannelID, runAt.UTC())
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, job)
	}
	s.recordAudit(ctx, workspaceID, "post.schedule_slots", postID.String())
	return jobs, nil
}

// checkPendingQuota enforces the per-workspace cap on not-yet-completed jobs
// (anti-abuse: bounds queued work and shared upstream-API load). adding is the
// number of jobs about to be created.
func (s *Service) checkPendingQuota(ctx context.Context, workspaceID uuid.UUID, adding int) error {
	count, err := s.pool.Queries().CountPendingJobsForWorkspace(ctx, workspaceID)
	if err != nil {
		return apperr.Internal(err)
	}
	if count+int64(adding) > maxPendingJobsPerWorkspace {
		return apperr.Validation("schedule_quota_exceeded",
			"this workspace has too many pending scheduled posts; wait for some to publish or cancel them")
	}
	return nil
}

// scheduleOne creates a job and enqueues its publish task, recording the task ID.
func (s *Service) scheduleOne(ctx context.Context, postID, channelID uuid.UUID, runAt time.Time) (Job, error) {
	row, err := s.pool.Queries().CreateScheduledJob(ctx, sqlc.CreateScheduledJobParams{
		PostID: postID, ChannelID: channelID, RunAt: tsFromTime(runAt), Status: StatusScheduled,
	})
	if err != nil {
		return Job{}, apperr.Internal(err)
	}
	taskID, err := s.enqueuer.EnqueuePublish(ctx, row.ID, runAt)
	if err != nil {
		// Mark the orphaned job failed so it isn't left dangling as "scheduled".
		_ = s.pool.Queries().SetScheduledJobStatus(ctx, sqlc.SetScheduledJobStatusParams{
			ID: row.ID, Status: StatusFailed, LastError: "enqueue failed: " + err.Error(), Attempts: 0,
		})
		return Job{}, apperr.Internal(err)
	}
	if err := s.pool.Queries().SetScheduledJobTaskID(ctx, sqlc.SetScheduledJobTaskIDParams{ID: row.ID, AsynqTaskID: taskID}); err != nil {
		return Job{}, apperr.Internal(err)
	}
	row.AsynqTaskID = taskID
	return toJob(row), nil
}

// Cancel cancels a scheduled (not-yet-run) job and removes its queued task. It
// errors if the job is not in a cancelable state (already publishing/published/
// failed/canceled) so the caller isn't told a no-op succeeded.
func (s *Service) Cancel(ctx context.Context, workspaceID, jobID uuid.UUID) error {
	job, err := s.ownedJob(ctx, workspaceID, jobID)
	if err != nil {
		return err
	}
	rows, err := s.pool.Queries().CancelScheduledJob(ctx, jobID)
	if err != nil {
		return apperr.Internal(err)
	}
	if rows == 0 {
		return apperr.Conflict("not_cancelable", "job is not in a cancelable state (already running or completed)")
	}
	if job.AsynqTaskID != "" {
		_ = s.enqueuer.Cancel(ctx, job.AsynqTaskID) // best-effort
	}
	s.recordAudit(ctx, workspaceID, "post.schedule_cancel", jobID.String())
	return nil
}

// Calendar returns the workspace's scheduled jobs within [from, to).
func (s *Service) Calendar(ctx context.Context, workspaceID uuid.UUID, from, to time.Time) ([]Job, error) {
	rows, err := s.pool.Queries().ListScheduledJobsInRange(ctx, sqlc.ListScheduledJobsInRangeParams{
		WorkspaceID: workspaceID, RunAt: tsFromTime(from.UTC()), RunAt_2: tsFromTime(to.UTC()),
	})
	if err != nil {
		return nil, apperr.Internal(err)
	}
	jobs := make([]Job, len(rows))
	for i, r := range rows {
		jobs[i] = toJob(r)
	}
	return jobs, nil
}

// ExecutionContext returns the channel and publish variant for a job, for the
// worker. The idempotency key is the job ID so a retried task never double-posts.
func (s *Service) ExecutionContext(ctx context.Context, jobID uuid.UUID) (uuid.UUID, publish.PostVariant, error) {
	job, err := s.pool.Queries().GetScheduledJob(ctx, jobID)
	if err != nil {
		return uuid.Nil, publish.PostVariant{}, err
	}
	v, err := s.pool.Queries().GetVariantByPostChannel(ctx, sqlc.GetVariantByPostChannelParams{
		PostID: job.PostID, ChannelID: job.ChannelID,
	})
	if err != nil {
		return uuid.Nil, publish.PostVariant{}, err
	}
	mediaRefs, err := s.loadMedia(ctx, v.MediaRefs)
	if err != nil {
		return uuid.Nil, publish.PostVariant{}, err
	}
	pv := publish.PostVariant{PostID: job.PostID, Text: v.Body, Media: mediaRefs, IdempotencyKey: jobID.String()}
	return job.ChannelID, pv, nil
}

// storedMediaRef is the subset of a variant's stored media_refs JSON the
// scheduler needs to load bytes for publishing.
type storedMediaRef struct {
	MediaID uuid.UUID `json:"media_id"`
}

// loadMedia downloads the bytes for each referenced media asset so the adapter
// can upload them. Errors are classified so the worker retries transient
// failures (storage outage) but not terminal ones (deleted asset, malformed
// refs): a brief storage blip must not permanently fail a scheduled post.
func (s *Service) loadMedia(ctx context.Context, mediaRefs []byte) ([]publish.MediaRef, error) {
	refs, err := parseMediaRefs(mediaRefs)
	if err != nil {
		return nil, publish.Terminal("invalid_media_refs", "stored media references are malformed", err)
	}
	if len(refs) == 0 {
		return nil, nil
	}
	if s.media == nil {
		// The variant references uploaded media but this process has no media
		// loader (storage unconfigured) — fail loudly instead of silently
		// publishing without the attachment. Retryable: configuring storage and
		// reprocessing should succeed.
		return nil, publish.Retryable("media_loader_unavailable", "media loader is not configured", nil)
	}
	out := make([]publish.MediaRef, 0, len(refs))
	for _, ref := range refs {
		kind, mime, data, err := s.media.OpenMedia(ctx, ref.MediaID)
		if err != nil {
			if apperr.KindOf(err) == apperr.KindNotFound {
				return nil, publish.Terminal("media_not_found", "attached media no longer exists", err)
			}
			return nil, publish.Retryable("media_load_failed", "could not load attached media", err)
		}
		// A presign failure only matters for URL-fetching platforms; the
		// adapter validates what it actually needs.
		url, _ := s.media.MediaURL(ctx, ref.MediaID, mediaURLTTL)
		out = append(out, publish.MediaRef{
			Kind: publish.MediaKind(kind), MIME: mime, Bytes: int64(len(data)), Data: data, URL: url,
		})
	}
	return out, nil
}

// parseMediaRefs decodes the stored media_refs JSON, keeping only entries that
// reference an uploaded asset (a non-nil media ID).
func parseMediaRefs(mediaRefs []byte) ([]storedMediaRef, error) {
	if len(mediaRefs) == 0 {
		return nil, nil
	}
	var all []storedMediaRef
	if err := json.Unmarshal(mediaRefs, &all); err != nil {
		return nil, err
	}
	out := make([]storedMediaRef, 0, len(all))
	for _, ref := range all {
		if ref.MediaID != uuid.Nil {
			out = append(out, ref)
		}
	}
	return out, nil
}

// Claim atomically transitions a job from scheduled to publishing (counting the
// attempt), reporting whether the claim succeeded. A job that was canceled or
// already handled is not scheduled and cannot be claimed, so the worker skips
// it — this is what makes cancellation actually prevent a publish even if the
// asynq task still fires.
func (s *Service) Claim(ctx context.Context, jobID uuid.UUID) (bool, error) {
	_, err := s.pool.Queries().ClaimScheduledJob(ctx, jobID)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// MarkPublished records a successful publish (the attempt was counted at Claim).
func (s *Service) MarkPublished(ctx context.Context, jobID uuid.UUID) error {
	return s.setStatus(ctx, jobID, StatusPublished, "", 0)
}

// MarkFailed records a terminal failure.
func (s *Service) MarkFailed(ctx context.Context, jobID uuid.UUID, cause string) error {
	return s.setStatus(ctx, jobID, StatusFailed, cause, 0)
}

// MarkRetry returns the job to scheduled so the worker's next attempt re-claims
// it. Only reachable after a successful Claim (status publishing), so it cannot
// resurrect a canceled job.
func (s *Service) MarkRetry(ctx context.Context, jobID uuid.UUID, cause string) error {
	return s.setStatus(ctx, jobID, StatusScheduled, cause, 0)
}

func (s *Service) setStatus(ctx context.Context, jobID uuid.UUID, status, cause string, attemptInc int32) error {
	return s.pool.Queries().SetScheduledJobStatus(ctx, sqlc.SetScheduledJobStatusParams{
		ID: jobID, Status: status, LastError: cause, Attempts: attemptInc,
	})
}

// postVariants loads a workspace-owned post's variants, erroring if the post is
// foreign/missing or has no variants.
func (s *Service) postVariants(ctx context.Context, workspaceID, postID uuid.UUID) ([]sqlc.PostVariant, error) {
	p, err := s.pool.Queries().GetPost(ctx, postID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperr.NotFound("post_not_found", "post not found")
		}
		return nil, apperr.Internal(err)
	}
	if p.WorkspaceID != workspaceID {
		return nil, apperr.NotFound("post_not_found", "post not found")
	}
	variants, err := s.pool.Queries().ListVariantsByPost(ctx, postID)
	if err != nil {
		return nil, apperr.Internal(err)
	}
	if len(variants) == 0 {
		return nil, apperr.Validation("no_variants", "post has no channel variants to schedule")
	}
	return variants, nil
}

// ownedJob loads a job and verifies its channel belongs to the workspace.
func (s *Service) ownedJob(ctx context.Context, workspaceID, jobID uuid.UUID) (sqlc.ScheduledJob, error) {
	job, err := s.pool.Queries().GetScheduledJob(ctx, jobID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return sqlc.ScheduledJob{}, apperr.NotFound("job_not_found", "scheduled job not found")
		}
		return sqlc.ScheduledJob{}, apperr.Internal(err)
	}
	if _, err := s.channels.PlatformFor(ctx, workspaceID, job.ChannelID); err != nil {
		return sqlc.ScheduledJob{}, apperr.NotFound("job_not_found", "scheduled job not found")
	}
	return job, nil
}

func (s *Service) recordAudit(ctx context.Context, workspaceID uuid.UUID, action, target string) {
	if s.audit == nil {
		return
	}
	ws := workspaceID
	_ = s.audit.Record(ctx, security.Event{WorkspaceID: &ws, Action: action, Target: target})
}

func toJob(r sqlc.ScheduledJob) Job {
	return Job{
		ID: r.ID, PostID: r.PostID, ChannelID: r.ChannelID, RunAt: r.RunAt.Time,
		Status: r.Status, Attempts: int(r.Attempts), LastError: r.LastError, CreatedAt: r.CreatedAt.Time,
	}
}

func tsFromTime(t time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: t, Valid: true}
}
