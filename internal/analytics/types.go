// Package analytics implements post-performance reporting: a poller fetches
// public metrics for published posts from each platform adapter and appends
// time-series snapshots, and workspace-scoped endpoints expose the latest
// values, per-post series, and a CSV export. Ingestion is platform-agnostic —
// each adapter returns named metrics, stored long-format in metric_snapshots.
package analytics

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/Akins20/postal/internal/publish"
)

// MetricsFetcher returns a published post's current platform metrics, handling
// the channel's token/refresh internally. *publish.Pipeline satisfies it; nil on
// the API server (only the worker polls).
type MetricsFetcher interface {
	FetchMetrics(ctx context.Context, channelID uuid.UUID, platformPostID string) ([]publish.Metric, error)
}

// PostMetrics is the latest value of every metric for one post on one channel.
// A compose-once post fans out to multiple channels, each reported separately.
type PostMetrics struct {
	PostID         uuid.UUID        `json:"post_id"`
	ChannelID      uuid.UUID        `json:"channel_id"`
	PlatformPostID string           `json:"platform_post_id"`
	Metrics        map[string]int64 `json:"metrics"`
	CapturedAt     time.Time        `json:"captured_at"`
}

// ChannelMetrics is the latest value of every metric for one post on a single
// channel (the per-channel breakdown for a post-detail view).
type ChannelMetrics struct {
	ChannelID      uuid.UUID        `json:"channel_id"`
	PlatformPostID string           `json:"platform_post_id"`
	Metrics        map[string]int64 `json:"metrics"`
	CapturedAt     time.Time        `json:"captured_at"`
}

// SeriesPoint is one datapoint in a metric's time series.
type SeriesPoint struct {
	Value      int64     `json:"value"`
	CapturedAt time.Time `json:"captured_at"`
}
