package analytics

import (
	"context"
	"encoding/csv"
	"io"
	"sort"
	"strconv"
	"time"

	"github.com/google/uuid"
)

// ExportCSV writes the workspace's latest metrics as CSV (one row per post per
// metric), most-recently-captured posts first. Columns are stable for clients.
func (s *Service) ExportCSV(ctx context.Context, workspaceID uuid.UUID, w io.Writer) error {
	overview, err := s.WorkspaceOverview(ctx, workspaceID)
	if err != nil {
		return err
	}
	cw := csv.NewWriter(w)
	if err := cw.Write([]string{"post_id", "channel_id", "platform_post_id", "metric", "value", "captured_at"}); err != nil {
		return err
	}
	for _, pm := range overview {
		for _, metric := range sortedKeys(pm.Metrics) {
			rec := []string{
				pm.PostID.String(), pm.ChannelID.String(), csvSafe(pm.PlatformPostID),
				csvSafe(metric), strconv.FormatInt(pm.Metrics[metric], 10), pm.CapturedAt.UTC().Format(time.RFC3339),
			}
			if err := cw.Write(rec); err != nil {
				return err
			}
		}
	}
	cw.Flush()
	return cw.Error()
}

// csvSafe neutralizes spreadsheet formula injection: a cell that a spreadsheet
// would evaluate (leading = + - @ or tab/CR) is prefixed with a single quote.
// Defense in depth for the free-text-ish fields (platform_post_id, metric) as
// new platform adapters land.
func csvSafe(s string) string {
	if s == "" {
		return s
	}
	switch s[0] {
	case '=', '+', '-', '@', '\t', '\r':
		return "'" + s
	}
	return s
}

// sortedKeys returns a metric map's keys in deterministic order.
func sortedKeys(m map[string]int64) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
