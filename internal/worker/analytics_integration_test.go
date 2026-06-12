package worker_test

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/Akins20/postal/internal/analytics"
	"github.com/Akins20/postal/internal/publish"
	"github.com/Akins20/postal/internal/worker"
)

// TestAnalytics_PollAndReport exercises the full analytics path: publish a post,
// configure platform metrics on the simulator, poll, and assert the snapshot is
// stored and surfaced through the reporting methods. It reuses the worker
// harness (PG/Redis/simulator + a seeded post).
func TestAnalytics_PollAndReport(t *testing.T) {
	h := setup(t)
	ctx := context.Background()

	// Publish the seeded post so a publish_results row (with platform_post_id) exists.
	jobs, err := h.sched.SchedulePost(ctx, h.wsID, h.postID, time.Now())
	if err != nil {
		t.Fatalf("SchedulePost: %v", err)
	}
	proc := worker.NewProcessor(h.sched, h.pipeline, h.channels, nil, nil, slog.Default(), nil)
	if err := proc.ProcessPublish(ctx, publishTask(t, jobs[0].ID)); err != nil {
		t.Fatalf("ProcessPublish: %v", err)
	}
	res, found, err := publish.NewStore(h.pool.Queries()).Find(ctx, jobs[0].ID.String())
	if err != nil || !found {
		t.Fatalf("publish result missing: found=%v err=%v", found, err)
	}

	// Poll dedup is keyed on (channel_id, platform_post_id); the seeded channel is
	// fresh per run, so the simulator's reused tweet ids don't collide.

	// Configure platform metrics for that post, then poll.
	h.sim.SetTweetMetrics(res.PlatformPostID, map[string]int64{"like_count": 42, "impression_count": 1000})
	svc := analytics.NewService(h.pool, h.pipeline, nil, nil)

	n, _, err := svc.PollMetrics(ctx)
	if err != nil {
		t.Fatalf("PollMetrics: %v", err)
	}
	if n < 1 {
		t.Fatalf("expected at least 1 post polled, got %d", n)
	}

	// Workspace overview reflects the ingested values.
	overview, err := svc.WorkspaceOverview(ctx, h.wsID)
	if err != nil {
		t.Fatalf("WorkspaceOverview: %v", err)
	}
	var found42 bool
	for _, pm := range overview {
		if pm.PostID == h.postID {
			if pm.Metrics["likes"] != 42 || pm.Metrics["impressions"] != 1000 {
				t.Fatalf("overview metrics = %+v, want likes=42 impressions=1000", pm.Metrics)
			}
			found42 = true
		}
	}
	if !found42 {
		t.Fatalf("post %s not present in overview", h.postID)
	}

	// Per-post latest is broken out per channel; our post's single channel has likes=42.
	latest, err := svc.LatestForPost(ctx, h.wsID, h.postID)
	if err != nil {
		t.Fatalf("LatestForPost: %v", err)
	}
	if len(latest) != 1 || latest[0].ChannelID != h.channelID || latest[0].Metrics["likes"] != 42 {
		t.Fatalf("LatestForPost = %+v, want one channel %s with likes=42", latest, h.channelID)
	}

	// The time series (post + channel + metric) has the datapoint.
	points, err := svc.Series(ctx, h.wsID, h.postID, h.channelID, "likes", time.Now().Add(-time.Hour), time.Now().Add(time.Hour))
	if err != nil {
		t.Fatalf("Series: %v", err)
	}
	if len(points) < 1 || points[len(points)-1].Value != 42 {
		t.Fatalf("series = %+v, want last value 42", points)
	}

	// CSV export includes the row.
	var buf bytes.Buffer
	if err := svc.ExportCSV(ctx, h.wsID, &buf); err != nil {
		t.Fatalf("ExportCSV: %v", err)
	}
	if !strings.Contains(buf.String(), "likes,42,") {
		t.Fatalf("CSV missing likes row:\n%s", buf.String())
	}

	// A second poll within the cadence window must not re-snapshot OUR post: it
	// was just captured, so it's no longer due. Assert its series doesn't grow.
	if _, _, err := svc.PollMetrics(ctx); err != nil {
		t.Fatalf("second PollMetrics: %v", err)
	}
	after, err := svc.Series(ctx, h.wsID, h.postID, h.channelID, "likes", time.Now().Add(-time.Hour), time.Now().Add(time.Hour))
	if err != nil {
		t.Fatalf("Series after second poll: %v", err)
	}
	if len(after) != len(points) {
		t.Errorf("post re-polled within cadence: series grew from %d to %d points", len(points), len(after))
	}
}
