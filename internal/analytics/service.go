package analytics

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"

	"github.com/Akins20/postal/internal/platform/apperr"
	"github.com/Akins20/postal/internal/platform/db"
	"github.com/Akins20/postal/internal/platform/db/sqlc"
	"github.com/Akins20/postal/internal/publish"
	"github.com/Akins20/postal/internal/security"
)

// Polling policy: poll posts published within the lookback window, at most once
// per cadence, in bounded batches with bounded fetch concurrency (anti-abuse +
// respects metered platform reads). Snapshots are retained for retentionWindow.
const (
	pollLookback    = 7 * 24 * time.Hour
	pollMinInterval = time.Hour
	pollBatch       = 200
	pollConcurrency = 8
	retentionWindow = 90 * 24 * time.Hour
)

// listLimit caps how many (post, channel) groups the workspace overview returns.
const listLimit = 500

// Service ingests post metrics (poller) and serves workspace-scoped reports.
type Service struct {
	pool    *db.Pool
	fetcher MetricsFetcher // nil on the API server (reporting only)
	audit   security.Recorder
	clock   func() time.Time
}

// NewService builds an analytics Service. fetcher may be nil (reporting-only,
// e.g. the API server); clock defaults to time.Now.
func NewService(pool *db.Pool, fetcher MetricsFetcher, audit security.Recorder, clock func() time.Time) *Service {
	if clock == nil {
		clock = time.Now
	}
	return &Service{pool: pool, fetcher: fetcher, audit: audit, clock: clock}
}

// PollMetrics fetches and stores a fresh metric snapshot for every published
// post due for polling, returning how many were snapshotted and how many failed.
// Fetches run with bounded concurrency. Every attempt records poll state (so a
// post isn't re-polled until the next cadence, and a deleted post is never
// re-polled); per-post failures are counted and skipped. err is only set when
// the batch can't be enumerated.
func (s *Service) PollMetrics(ctx context.Context) (polled, failed int, err error) {
	if s.fetcher == nil {
		return 0, 0, nil
	}
	now := s.clock()
	// Retention: bound table growth before adding this sweep's rows.
	if perr := s.pool.Queries().PruneSnapshotsBefore(ctx, db.Timestamptz(now.Add(-retentionWindow))); perr != nil {
		return 0, 0, apperr.Internal(perr)
	}
	rows, err := s.pool.Queries().ListPostsDueForMetrics(ctx, sqlc.ListPostsDueForMetricsParams{
		PublishedAt:  db.Timestamptz(now.Add(-pollLookback)),
		LastPolledAt: db.Timestamptz(now.Add(-pollMinInterval)),
		Limit:        pollBatch,
	})
	if err != nil {
		return 0, 0, apperr.Internal(err)
	}

	var polledN, failedN int64
	sem := make(chan struct{}, pollConcurrency)
	var wg sync.WaitGroup
	for _, row := range rows {
		wg.Add(1)
		sem <- struct{}{}
		go func(row sqlc.ListPostsDueForMetricsRow) {
			defer wg.Done()
			defer func() { <-sem }()
			if s.pollOne(ctx, row, now) {
				atomic.AddInt64(&polledN, 1)
			} else {
				atomic.AddInt64(&failedN, 1)
			}
		}(row)
	}
	wg.Wait()
	return int(polledN), int(failedN), nil
}

// pollOne fetches and stores one post's metrics, recording poll state in the
// same transaction. It reports success. A terminal fetch error (the post is gone
// at the platform) marks the post done so it's never polled again.
func (s *Service) pollOne(ctx context.Context, row sqlc.ListPostsDueForMetricsRow, now time.Time) bool {
	metrics, ferr := s.fetcher.FetchMetrics(ctx, row.ChannelID, row.PlatformPostID)
	if ferr != nil {
		// Record the attempt so a transient failure isn't retried until the next
		// cadence, and a terminal failure (deleted post) stops polling for good.
		_ = s.recordPollState(ctx, row, now, isTerminal(ferr))
		return false
	}
	if rerr := s.storeSnapshot(ctx, row, metrics, now); rerr != nil {
		return false
	}
	return true
}

// storeSnapshot writes all of a post's metrics plus its poll state atomically, so
// the dedup never sees a partial capture as complete.
func (s *Service) storeSnapshot(ctx context.Context, row sqlc.ListPostsDueForMetricsRow, metrics []publish.Metric, now time.Time) error {
	if len(metrics) == 0 {
		return s.recordPollState(ctx, row, now, false)
	}
	at := db.Timestamptz(now)
	params := make([]sqlc.InsertMetricSnapshotsParams, len(metrics))
	for i, m := range metrics {
		params[i] = sqlc.InsertMetricSnapshotsParams{
			WorkspaceID: row.WorkspaceID, ChannelID: row.ChannelID, PostID: row.PostID,
			PlatformPostID: row.PlatformPostID, Metric: m.Name, Value: m.Value, CapturedAt: at,
		}
	}
	return s.pool.WithTx(ctx, func(q *sqlc.Queries) error {
		if _, err := q.InsertMetricSnapshots(ctx, params); err != nil {
			return err
		}
		return q.UpsertPollState(ctx, sqlc.UpsertPollStateParams{
			ChannelID: row.ChannelID, PlatformPostID: row.PlatformPostID, LastPolledAt: at, Done: false,
		})
	})
}

// recordPollState upserts the poll attempt (last_polled_at / done) for a post.
func (s *Service) recordPollState(ctx context.Context, row sqlc.ListPostsDueForMetricsRow, now time.Time, done bool) error {
	return s.pool.Queries().UpsertPollState(ctx, sqlc.UpsertPollStateParams{
		ChannelID: row.ChannelID, PlatformPostID: row.PlatformPostID, LastPolledAt: db.Timestamptz(now), Done: done,
	})
}

// isTerminal reports whether a fetch error is terminal (the post no longer
// exists / will never succeed) versus transient.
func isTerminal(err error) bool {
	var ae *publish.Error
	if errors.As(err, &ae) {
		return ae.Class == publish.ClassTerminal
	}
	return false
}

// WorkspaceOverview returns the latest value of every metric for the most
// recently active (post, channel) pairs in the workspace.
func (s *Service) WorkspaceOverview(ctx context.Context, workspaceID uuid.UUID) ([]PostMetrics, error) {
	rows, err := s.pool.Queries().LatestMetricsForWorkspace(ctx, sqlc.LatestMetricsForWorkspaceParams{
		WorkspaceID: workspaceID, Limit: listLimit,
	})
	if err != nil {
		return nil, apperr.Internal(err)
	}
	// Rows arrive grouped by (post, channel) in recency order; preserve it.
	type key struct {
		post    uuid.UUID
		channel uuid.UUID
	}
	byGroup := make(map[key]*PostMetrics)
	order := make([]key, 0)
	for _, r := range rows {
		if r.PostID == nil {
			continue
		}
		k := key{post: *r.PostID, channel: r.ChannelID}
		pm, ok := byGroup[k]
		if !ok {
			pm = &PostMetrics{PostID: *r.PostID, ChannelID: r.ChannelID, PlatformPostID: r.PlatformPostID, Metrics: map[string]int64{}}
			byGroup[k] = pm
			order = append(order, k)
		}
		pm.Metrics[r.Metric] = r.Value
		if r.CapturedAt.Time.After(pm.CapturedAt) {
			pm.CapturedAt = r.CapturedAt.Time
		}
	}
	out := make([]PostMetrics, 0, len(order))
	for _, k := range order {
		out = append(out, *byGroup[k])
	}
	return out, nil
}

// LatestForPost returns the latest value of every metric for one post, broken
// out per channel it was published to (workspace-scoped).
func (s *Service) LatestForPost(ctx context.Context, workspaceID, postID uuid.UUID) ([]ChannelMetrics, error) {
	pid := postID
	rows, err := s.pool.Queries().LatestMetricsForPost(ctx, sqlc.LatestMetricsForPostParams{
		WorkspaceID: workspaceID, PostID: &pid,
	})
	if err != nil {
		return nil, apperr.Internal(err)
	}
	byChannel := make(map[uuid.UUID]*ChannelMetrics)
	order := make([]uuid.UUID, 0)
	for _, r := range rows {
		cm, ok := byChannel[r.ChannelID]
		if !ok {
			cm = &ChannelMetrics{ChannelID: r.ChannelID, PlatformPostID: r.PlatformPostID, Metrics: map[string]int64{}}
			byChannel[r.ChannelID] = cm
			order = append(order, r.ChannelID)
		}
		cm.Metrics[r.Metric] = r.Value
		if r.CapturedAt.Time.After(cm.CapturedAt) {
			cm.CapturedAt = r.CapturedAt.Time
		}
	}
	out := make([]ChannelMetrics, 0, len(order))
	for _, id := range order {
		out = append(out, *byChannel[id])
	}
	return out, nil
}

// Series returns the time series of one metric for one post on one channel
// within [from, to].
func (s *Service) Series(ctx context.Context, workspaceID, postID, channelID uuid.UUID, metric string, from, to time.Time) ([]SeriesPoint, error) {
	if metric == "" {
		return nil, apperr.Validation("missing_metric", "a metric name is required")
	}
	if channelID == uuid.Nil {
		return nil, apperr.Validation("missing_channel", "a channel_id is required")
	}
	pid := postID
	rows, err := s.pool.Queries().MetricSeries(ctx, sqlc.MetricSeriesParams{
		WorkspaceID: workspaceID, PostID: &pid, ChannelID: channelID, Metric: metric,
		CapturedAt: db.Timestamptz(from), CapturedAt_2: db.Timestamptz(to),
	})
	if err != nil {
		return nil, apperr.Internal(err)
	}
	out := make([]SeriesPoint, len(rows))
	for i, r := range rows {
		out[i] = SeriesPoint{Value: r.Value, CapturedAt: r.CapturedAt.Time}
	}
	return out, nil
}
