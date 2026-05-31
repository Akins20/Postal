-- name: InsertMetricSnapshots :copyfrom
-- Batched insert of all metrics for one poll (atomic, single round-trip).
INSERT INTO metric_snapshots (workspace_id, channel_id, post_id, platform_post_id, metric, value, captured_at)
VALUES ($1, $2, $3, $4, $5, $6, $7);

-- name: UpsertPollState :exec
-- Record a poll attempt for a (channel, platform post). done is sticky: once a
-- post is gone at the platform it stays done so it's never polled again.
INSERT INTO metric_poll_state (channel_id, platform_post_id, last_polled_at, done)
VALUES ($1, $2, $3, $4)
ON CONFLICT (channel_id, platform_post_id)
DO UPDATE SET last_polled_at = EXCLUDED.last_polled_at,
              done = metric_poll_state.done OR EXCLUDED.done;

-- name: PruneSnapshotsBefore :exec
-- Retention: drop snapshots older than the cutoff to bound table growth.
DELETE FROM metric_snapshots WHERE captured_at < $1;

-- name: ListPostsDueForMetrics :many
-- Published posts to poll: published within the lookback window and not polled
-- since the cutoff (dedup is per (channel, platform post) via poll-state, so a
-- collision on platform_post_id across channels can't suppress another's poll,
-- and a post marked done is excluded permanently).
SELECT c.workspace_id, pr.channel_id, pr.post_id, pr.platform_post_id, c.platform
FROM publish_results pr
JOIN channels c ON c.id = pr.channel_id
WHERE pr.published_at > $1
  AND NOT EXISTS (
    SELECT 1 FROM metric_poll_state ps
    WHERE ps.channel_id = pr.channel_id
      AND ps.platform_post_id = pr.platform_post_id
      AND (ps.done OR ps.last_polled_at > $2)
  )
ORDER BY pr.published_at DESC
LIMIT $3;

-- name: LatestMetricsForWorkspace :many
-- The most recent value of every metric for the N most-recently-active
-- (post, channel) pairs in the workspace. The LIMIT is applied in SQL (by
-- recency) so a large workspace never materializes its whole history.
WITH latest AS (
    SELECT DISTINCT ON (post_id, channel_id, metric)
        post_id, channel_id, platform_post_id, metric, value, captured_at
    FROM metric_snapshots
    WHERE workspace_id = $1 AND post_id IS NOT NULL
    ORDER BY post_id, channel_id, metric, captured_at DESC
),
groups AS (
    SELECT post_id, channel_id, MAX(captured_at) AS latest_at
    FROM latest
    GROUP BY post_id, channel_id
    ORDER BY latest_at DESC
    LIMIT $2
)
SELECT l.post_id, l.channel_id, l.platform_post_id, l.metric, l.value, l.captured_at
FROM latest l
JOIN groups g ON g.post_id = l.post_id AND g.channel_id = l.channel_id
ORDER BY g.latest_at DESC, l.channel_id, l.metric;

-- name: LatestMetricsForPost :many
-- The most recent value of every metric for one post, per channel it was
-- published to (workspace-scoped).
SELECT DISTINCT ON (channel_id, metric)
    channel_id, platform_post_id, metric, value, captured_at
FROM metric_snapshots
WHERE workspace_id = $1 AND post_id = $2
ORDER BY channel_id, metric, captured_at DESC;

-- name: MetricSeries :many
-- The time series of one metric for one post on one channel within a window.
SELECT value, captured_at
FROM metric_snapshots
WHERE workspace_id = $1 AND post_id = $2 AND channel_id = $3 AND metric = $4
  AND captured_at >= $5 AND captured_at <= $6
ORDER BY captured_at;
