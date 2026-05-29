package twitter

import (
	"context"
	"fmt"

	"github.com/Akins20/postal/internal/channel"
	"github.com/Akins20/postal/internal/publish"
)

// FetchMetrics returns public metrics for a published post via the post-lookup
// endpoint with tweet.fields=public_metrics.
func (a *Adapter) FetchMetrics(ctx context.Context, token channel.Token, platformPostID string) ([]publish.Metric, error) {
	url := fmt.Sprintf("%s/2/tweets/%s?tweet.fields=public_metrics", a.cfg.APIBaseURL, platformPostID)

	var resp struct {
		Data struct {
			PublicMetrics struct {
				LikeCount       int64 `json:"like_count"`
				RetweetCount    int64 `json:"retweet_count"`
				ReplyCount      int64 `json:"reply_count"`
				QuoteCount      int64 `json:"quote_count"`
				ImpressionCount int64 `json:"impression_count"`
				BookmarkCount   int64 `json:"bookmark_count"`
			} `json:"public_metrics"`
		} `json:"data"`
	}
	if err := a.getJSON(ctx, url, token.AccessToken, &resp); err != nil {
		return nil, err
	}

	m := resp.Data.PublicMetrics
	return []publish.Metric{
		{Name: "likes", Value: m.LikeCount},
		{Name: "reposts", Value: m.RetweetCount},
		{Name: "replies", Value: m.ReplyCount},
		{Name: "quotes", Value: m.QuoteCount},
		{Name: "impressions", Value: m.ImpressionCount},
		{Name: "bookmarks", Value: m.BookmarkCount},
	}, nil
}
