package analytics_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/Akins20/postal/internal/analytics"
	"github.com/Akins20/postal/internal/platform/db"
	"github.com/Akins20/postal/internal/platform/db/sqlc"
)

// TestReporting_PerChannelBreakout verifies that a compose-once post published to
// two channels reports each channel's metrics separately (not collapsed into
// one), exercising the workspace overview, per-post breakdown, and series.
func TestReporting_PerChannelBreakout(t *testing.T) {
	dsn := os.Getenv("POSTAL_DATABASE_URL")
	if dsn == "" {
		t.Skip("POSTAL_DATABASE_URL not set; skipping integration test")
	}
	ctx := context.Background()
	pool, err := db.Connect(ctx, dsn)
	if err != nil {
		t.Skipf("postgres unreachable: %v", err)
	}
	t.Cleanup(pool.Close)
	q := pool.Queries()

	user, err := q.CreateUser(ctx, sqlc.CreateUserParams{Email: "ana-" + uuid.NewString() + "@example.com", PasswordHash: "x"})
	if err != nil {
		t.Fatalf("user: %v", err)
	}
	ws, err := q.CreateWorkspace(ctx, sqlc.CreateWorkspaceParams{Name: "Ana", OwnerUserID: user.ID})
	if err != nil {
		t.Fatalf("workspace: %v", err)
	}
	chA := seedChannel(t, ctx, q, ws.ID, user.ID)
	chB := seedChannel(t, ctx, q, ws.ID, user.ID)
	post, err := q.CreatePost(ctx, sqlc.CreatePostParams{WorkspaceID: ws.ID, AuthorUserID: &user.ID, Status: "draft"})
	if err != nil {
		t.Fatalf("post: %v", err)
	}

	// Channel A: likes=10, impressions=100. Channel B: likes=4. Same post.
	now := time.Now().UTC()
	insertSnap(t, ctx, q, ws.ID, chA, post.ID, "ppA", now, map[string]int64{"likes": 10, "impressions": 100})
	insertSnap(t, ctx, q, ws.ID, chB, post.ID, "ppB", now, map[string]int64{"likes": 4})

	svc := analytics.NewService(pool, nil, nil, nil)

	// Overview: two entries (one per channel), each with that channel's metrics.
	overview, err := svc.WorkspaceOverview(ctx, ws.ID)
	if err != nil {
		t.Fatalf("WorkspaceOverview: %v", err)
	}
	got := map[uuid.UUID]int64{}
	for _, pm := range overview {
		if pm.PostID == post.ID {
			got[pm.ChannelID] = pm.Metrics["likes"]
		}
	}
	if got[chA] != 10 || got[chB] != 4 {
		t.Fatalf("overview per-channel likes = {A:%d B:%d}, want {A:10 B:4}; entries=%+v", got[chA], got[chB], overview)
	}

	// Per-post breakdown returns both channels.
	latest, err := svc.LatestForPost(ctx, ws.ID, post.ID)
	if err != nil {
		t.Fatalf("LatestForPost: %v", err)
	}
	if len(latest) != 2 {
		t.Fatalf("LatestForPost channels = %d, want 2: %+v", len(latest), latest)
	}

	// Series is scoped to one channel.
	pts, err := svc.Series(ctx, ws.ID, post.ID, chA, "likes", now.Add(-time.Hour), now.Add(time.Hour))
	if err != nil {
		t.Fatalf("Series: %v", err)
	}
	if len(pts) != 1 || pts[0].Value != 10 {
		t.Fatalf("series for channel A = %+v, want one point value 10", pts)
	}
}

func seedChannel(t *testing.T, ctx context.Context, q *sqlc.Queries, wsID, userID uuid.UUID) uuid.UUID {
	t.Helper()
	ch, err := q.CreateChannel(ctx, sqlc.CreateChannelParams{
		WorkspaceID: wsID, Platform: "twitter", PlatformAccountID: "acct-" + uuid.NewString(),
		Handle: "@x", DisplayName: "X", ConnectedBy: &userID,
	})
	if err != nil {
		t.Fatalf("channel: %v", err)
	}
	return ch.ID
}

func insertSnap(t *testing.T, ctx context.Context, q *sqlc.Queries, wsID, chID, postID uuid.UUID, ppID string, at time.Time, metrics map[string]int64) {
	t.Helper()
	pid := postID
	params := make([]sqlc.InsertMetricSnapshotsParams, 0, len(metrics))
	for name, v := range metrics {
		params = append(params, sqlc.InsertMetricSnapshotsParams{
			WorkspaceID: wsID, ChannelID: chID, PostID: &pid, PlatformPostID: ppID,
			Metric: name, Value: v, CapturedAt: db.Timestamptz(at),
		})
	}
	if _, err := q.InsertMetricSnapshots(ctx, params); err != nil {
		t.Fatalf("insert snapshots: %v", err)
	}
}
