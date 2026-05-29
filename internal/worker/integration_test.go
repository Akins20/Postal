package worker_test

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	goredis "github.com/redis/go-redis/v9"

	"github.com/Akins20/postal/internal/channel"
	"github.com/Akins20/postal/internal/platform/db"
	"github.com/Akins20/postal/internal/platform/db/sqlc"
	"github.com/Akins20/postal/internal/publish"
	twittersim "github.com/Akins20/postal/internal/publish/simulator/twitter"
	"github.com/Akins20/postal/internal/publish/twitter"
	"github.com/Akins20/postal/internal/schedule"
	"github.com/Akins20/postal/internal/security"
	"github.com/Akins20/postal/internal/worker"
	"github.com/Akins20/postal/internal/workspace"
)

// fakeEnqueuer records nothing real — the test invokes the processor directly
// rather than running the asynq server, so scheduling just needs a task ID.
type fakeEnqueuer struct{}

func (fakeEnqueuer) EnqueuePublish(_ context.Context, jobID uuid.UUID, _ time.Time) (string, error) {
	return "task-" + jobID.String(), nil
}
func (fakeEnqueuer) Cancel(context.Context, string) error { return nil }

// harness wires the full publish path against a simulator + real PG/Redis and
// seeds a workspace/channel/credential/post, returning the schedule service,
// pipeline, channel service, and the seeded post + workspace IDs.
type harness struct {
	sched     *schedule.Service
	pipeline  *publish.Pipeline
	channels  *channel.Service
	pool      *db.Pool
	sim       *twittersim.Server
	wsID      uuid.UUID
	postID    uuid.UUID
	channelID uuid.UUID
}

func setup(t *testing.T) *harness {
	t.Helper()
	dsn, addr := os.Getenv("POSTAL_DATABASE_URL"), os.Getenv("POSTAL_REDIS_ADDR")
	if dsn == "" || addr == "" {
		t.Skip("POSTAL_DATABASE_URL/POSTAL_REDIS_ADDR not set; skipping integration test")
	}
	ctx := context.Background()
	pool, err := db.Connect(ctx, dsn)
	if err != nil {
		t.Skipf("postgres unreachable: %v", err)
	}
	t.Cleanup(pool.Close)
	rdb := goredis.NewClient(&goredis.Options{Addr: addr, Password: os.Getenv("POSTAL_REDIS_PASSWORD")})
	if err := rdb.Ping(ctx).Err(); err != nil {
		t.Skipf("redis unreachable: %v", err)
	}
	t.Cleanup(func() { _ = rdb.Close() })

	sim := twittersim.New()
	t.Cleanup(sim.Close)
	enc, err := security.NewEncryptorFromSpec(base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{9}, 32)))
	if err != nil {
		t.Fatalf("encryptor: %v", err)
	}
	adapter := twitter.New(twitter.Config{ClientID: "c", RedirectURI: "https://app/cb", APIBaseURL: sim.URL(), AuthBaseURL: sim.URL()})

	wsID, postID, channelID := seed(t, ctx, pool, enc, adapter, sim)

	wsSvc := workspace.NewService(pool, nil, nil)
	channels := channel.NewService(pool, channel.NewRegistry(adapter), enc, rdb, wsSvc, nil, nil)
	pipeline := publish.NewPipeline(channels, publish.NewStore(pool.Queries()), []publish.Adapter{adapter})
	sched := schedule.NewService(pool, channels, fakeEnqueuer{}, nil, nil)
	return &harness{sched: sched, pipeline: pipeline, channels: channels, pool: pool, sim: sim, wsID: wsID, postID: postID, channelID: channelID}
}

// seed creates a user, workspace, channel with a simulator-valid encrypted
// credential, and a post variant; returns workspace/post/channel IDs.
func seed(t *testing.T, ctx context.Context, pool *db.Pool, enc *security.Encryptor, adapter *twitter.Adapter, _ *twittersim.Server) (uuid.UUID, uuid.UUID, uuid.UUID) {
	t.Helper()
	q := pool.Queries()
	user, err := q.CreateUser(ctx, sqlc.CreateUserParams{Email: "wk-" + uuid.NewString() + "@example.com", PasswordHash: "x"})
	if err != nil {
		t.Fatalf("user: %v", err)
	}
	ws, err := q.CreateWorkspace(ctx, sqlc.CreateWorkspaceParams{Name: "Sched", OwnerUserID: user.ID})
	if err != nil {
		t.Fatalf("workspace: %v", err)
	}
	ch, err := q.CreateChannel(ctx, sqlc.CreateChannelParams{
		WorkspaceID: ws.ID, Platform: "twitter", PlatformAccountID: "acct-" + uuid.NewString(),
		Handle: "@x", DisplayName: "X", ConnectedBy: &user.ID,
	})
	if err != nil {
		t.Fatalf("channel: %v", err)
	}

	// Obtain a simulator-valid token and store it encrypted as the credential.
	tok, err := adapter.ExchangeCode(ctx, "seed-code", "verifier")
	if err != nil {
		t.Fatalf("exchange: %v", err)
	}
	access, _ := enc.Seal([]byte(tok.AccessToken))
	refresh, _ := enc.Seal([]byte(tok.RefreshToken))
	if err := q.UpsertChannelCredential(ctx, sqlc.UpsertChannelCredentialParams{
		ChannelID: ch.ID, EncryptedAccessToken: access, EncryptedRefreshToken: refresh,
		Scopes: tok.Scopes, KeyVersion: int32(enc.CurrentVersion()), //nolint:gosec // small version counter
	}); err != nil {
		t.Fatalf("credential: %v", err)
	}

	p, err := q.CreatePost(ctx, sqlc.CreatePostParams{WorkspaceID: ws.ID, AuthorUserID: &user.ID, Status: "draft"})
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	if _, err := q.CreatePostVariant(ctx, sqlc.CreatePostVariantParams{
		PostID: p.ID, ChannelID: ch.ID, Body: "scheduled hello", MediaRefs: []byte("[]"), PlatformOptions: []byte("{}"),
	}); err != nil {
		t.Fatalf("variant: %v", err)
	}
	return ws.ID, p.ID, ch.ID
}

func publishTask(t *testing.T, jobID uuid.UUID) *asynq.Task {
	t.Helper()
	return asynq.NewTask(worker.TypePublish, []byte(fmt.Sprintf(`{"job_id":%q}`, jobID.String())))
}

func TestWorker_ScheduleThenProcess_Publishes(t *testing.T) {
	h := setup(t)
	ctx := context.Background()

	jobs, err := h.sched.SchedulePost(ctx, h.wsID, h.postID, time.Now())
	if err != nil {
		t.Fatalf("SchedulePost: %v", err)
	}
	if len(jobs) != 1 {
		t.Fatalf("expected 1 job, got %d", len(jobs))
	}
	jobID := jobs[0].ID

	proc := worker.NewProcessor(h.sched, h.pipeline, h.channels, slog.Default(), nil)
	if err := proc.ProcessPublish(ctx, publishTask(t, jobID)); err != nil {
		t.Fatalf("ProcessPublish: %v", err)
	}

	// The simulator received exactly one tweet.
	if h.sim.TweetCount() != 1 {
		t.Errorf("simulator tweet count = %d, want 1", h.sim.TweetCount())
	}
	// The job is marked published.
	job, _ := h.pool.Queries().GetScheduledJob(ctx, jobID)
	if job.Status != schedule.StatusPublished {
		t.Errorf("job status = %q, want published", job.Status)
	}
	// A publish_result is recorded under the job-ID idempotency key.
	res, found, err := publish.NewStore(h.pool.Queries()).Find(ctx, jobID.String())
	if err != nil || !found || res.PlatformPostID == "" {
		t.Errorf("publish_result not recorded: found=%v err=%v", found, err)
	}

	// Re-delivery of a completed job is not claimable (status=published) → the
	// worker skips it (returns a SkipRetry error) and does NOT publish again.
	if err := proc.ProcessPublish(ctx, publishTask(t, jobID)); err == nil {
		t.Error("re-processing a published job should be skipped (not claimable)")
	}
	if h.sim.TweetCount() != 1 {
		t.Errorf("idempotency violated: tweet count = %d, want 1", h.sim.TweetCount())
	}
}

func TestWorker_CanceledJobNotPublished(t *testing.T) {
	h := setup(t)
	ctx := context.Background()
	proc := worker.NewProcessor(h.sched, h.pipeline, h.channels, slog.Default(), nil)

	jobs, err := h.sched.SchedulePost(ctx, h.wsID, h.postID, time.Now())
	if err != nil {
		t.Fatalf("SchedulePost: %v", err)
	}
	// Cancel before the task fires, then deliver the task anyway (asynq race).
	if err := h.sched.Cancel(ctx, h.wsID, jobs[0].ID); err != nil {
		t.Fatalf("Cancel: %v", err)
	}
	if err := proc.ProcessPublish(ctx, publishTask(t, jobs[0].ID)); err == nil {
		t.Error("a canceled job should not be claimable/published")
	}
	if h.sim.TweetCount() != 0 {
		t.Errorf("canceled job was published: tweet count = %d, want 0", h.sim.TweetCount())
	}
}

func TestWorker_DuplicateContent_Terminal(t *testing.T) {
	h := setup(t)
	ctx := context.Background()
	proc := worker.NewProcessor(h.sched, h.pipeline, h.channels, slog.Default(), nil)

	// First publish succeeds.
	jobs, _ := h.sched.SchedulePost(ctx, h.wsID, h.postID, time.Now())
	if err := proc.ProcessPublish(ctx, publishTask(t, jobs[0].ID)); err != nil {
		t.Fatalf("first publish: %v", err)
	}

	// Schedule the SAME post again (same text) -> duplicate at the platform ->
	// terminal -> job failed, and the handler returns a SkipRetry-wrapped error.
	jobs2, _ := h.sched.SchedulePost(ctx, h.wsID, h.postID, time.Now())
	err := proc.ProcessPublish(ctx, publishTask(t, jobs2[0].ID))
	if err == nil {
		t.Fatal("duplicate publish should error (SkipRetry)")
	}
	job, _ := h.pool.Queries().GetScheduledJob(ctx, jobs2[0].ID)
	if job.Status != schedule.StatusFailed {
		t.Errorf("duplicate job status = %q, want failed", job.Status)
	}
}

func TestSchedule_CancelMarksCanceled(t *testing.T) {
	h := setup(t)
	ctx := context.Background()
	jobs, err := h.sched.SchedulePost(ctx, h.wsID, h.postID, time.Now().Add(time.Hour))
	if err != nil {
		t.Fatalf("SchedulePost: %v", err)
	}
	if err := h.sched.Cancel(ctx, h.wsID, jobs[0].ID); err != nil {
		t.Fatalf("Cancel: %v", err)
	}
	job, _ := h.pool.Queries().GetScheduledJob(ctx, jobs[0].ID)
	if job.Status != schedule.StatusCanceled {
		t.Errorf("status = %q, want canceled", job.Status)
	}
}

func TestSchedule_NextOpenSlot(t *testing.T) {
	h := setup(t)
	ctx := context.Background()
	// A Monday 09:00 UTC slot.
	if _, err := h.sched.CreateSlot(ctx, h.wsID, h.channelID, int(time.Monday), "09:00", "UTC"); err != nil {
		t.Fatalf("CreateSlot: %v", err)
	}
	from := time.Date(2026, 6, 2, 12, 0, 0, 0, time.UTC) // a Tuesday
	next, err := h.sched.NextOpenSlot(ctx, h.channelID, from)
	if err != nil {
		t.Fatalf("NextOpenSlot: %v", err)
	}
	if next.Weekday() != time.Monday || next.Hour() != 9 || !next.After(from) {
		t.Errorf("next slot = %s, want next Monday 09:00 UTC after %s", next, from)
	}
}
