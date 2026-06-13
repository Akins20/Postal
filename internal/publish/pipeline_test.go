package publish_test

import (
	"context"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/Akins20/postal/internal/channel"
	"github.com/Akins20/postal/internal/publish"
	twittersim "github.com/Akins20/postal/internal/publish/simulator/twitter"
	"github.com/Akins20/postal/internal/publish/twitter"
)

// fakeChannels is an in-memory Channels: it holds the current token and, on
// Refresh, asks the adapter for a genuinely new token (as the real channel
// service would), simulating the store without a database.
type fakeChannels struct {
	platform     string
	adapter      *twitter.Adapter
	token        channel.Token
	refreshTok   string
	refreshCount int
}

func (f *fakeChannels) PublishContext(context.Context, uuid.UUID) (string, channel.Token, error) {
	return f.platform, f.token, nil
}

func (f *fakeChannels) Refresh(ctx context.Context, _ uuid.UUID) (channel.Token, error) {
	f.refreshCount++
	tok, err := f.adapter.RefreshToken(ctx, f.refreshTok)
	if err != nil {
		return channel.Token{}, err
	}
	f.token = *tok
	f.refreshTok = tok.RefreshToken
	return *tok, nil
}

// memResults is an in-memory Results store for idempotency assertions.
type memResults struct {
	mu sync.Mutex
	m  map[string]*publish.Result
}

func newMemResults() *memResults { return &memResults{m: map[string]*publish.Result{}} }

func (r *memResults) Find(_ context.Context, key string) (*publish.Result, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	res, ok := r.m[key]
	return res, ok, nil
}

func (r *memResults) Record(_ context.Context, _, _ uuid.UUID, key string, res *publish.Result) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.m[key] = res
	return nil
}

// harness wires an adapter against a simulator with a fresh authenticated token.
func harness(t *testing.T) (*twitter.Adapter, *twittersim.Server, *fakeChannels, *memResults) {
	t.Helper()
	sim := twittersim.New()
	t.Cleanup(sim.Close)
	a := twitter.New(twitter.Config{ClientID: "c", RedirectURI: "https://app/cb", APIBaseURL: sim.URL(), AuthBaseURL: sim.URL()})
	tok, err := a.ExchangeCode(context.Background(), "code", "verifier", "")
	if err != nil {
		t.Fatalf("exchange: %v", err)
	}
	return a, sim, &fakeChannels{platform: "twitter", adapter: a, token: *tok, refreshTok: tok.RefreshToken}, newMemResults()
}

// noSleep is a no-op sleeper so retry tests don't actually wait.
func noSleep(context.Context, time.Duration) error { return nil }

func newPipeline(ch publish.Channels, res publish.Results, a *twitter.Adapter) *publish.Pipeline {
	return publish.NewPipeline(ch, res, []publish.Adapter{a}, publish.WithSleeper(noSleep))
}

func TestPipeline_FetchMetrics(t *testing.T) {
	a, sim, ch, res := harness(t)
	p := newPipeline(ch, res, a)
	ctx := context.Background()

	// Publish a post so the simulator has a tweet to look up.
	out, err := p.Publish(ctx, uuid.New(), publish.PostVariant{Text: "metric me", IdempotencyKey: "m1"})
	if err != nil {
		t.Fatalf("Publish: %v", err)
	}
	sim.SetTweetMetrics(out.PlatformPostID, map[string]int64{"like_count": 7})

	metrics, err := p.FetchMetrics(ctx, uuid.New(), out.PlatformPostID)
	if err != nil {
		t.Fatalf("FetchMetrics: %v", err)
	}
	if metricValue(metrics, "likes") != 7 {
		t.Fatalf("likes = %d, want 7", metricValue(metrics, "likes"))
	}
}

func TestPipeline_FetchMetrics_RefreshesExpiredToken(t *testing.T) {
	a, sim, ch, res := harness(t)
	p := newPipeline(ch, res, a)
	ctx := context.Background()

	out, err := p.Publish(ctx, uuid.New(), publish.PostVariant{Text: "refresh me", IdempotencyKey: "m2"})
	if err != nil {
		t.Fatalf("Publish: %v", err)
	}
	// Expire all tokens: the first metrics call 401s, the pipeline refreshes once
	// and retries successfully.
	sim.ExpireAccessTokens()

	if _, err := p.FetchMetrics(ctx, uuid.New(), out.PlatformPostID); err != nil {
		t.Fatalf("FetchMetrics after expiry: %v", err)
	}
	if ch.refreshCount != 1 {
		t.Errorf("refresh count = %d, want exactly 1", ch.refreshCount)
	}
}

// metricValue returns the value of the named metric, or -1.
func metricValue(metrics []publish.Metric, name string) int64 {
	for _, m := range metrics {
		if m.Name == name {
			return m.Value
		}
	}
	return -1
}

func TestPipeline_HappyPath(t *testing.T) {
	a, sim, ch, res := harness(t)
	p := newPipeline(ch, res, a)

	out, err := p.Publish(context.Background(), uuid.New(), publish.PostVariant{Text: "hi", IdempotencyKey: "k1"})
	if err != nil {
		t.Fatalf("Publish: %v", err)
	}
	if out.PlatformPostID == "" || sim.TweetCount() != 1 {
		t.Error("expected one published tweet")
	}
}

func TestPipeline_Idempotency_NoDoublePost(t *testing.T) {
	a, sim, ch, res := harness(t)
	p := newPipeline(ch, res, a)
	chID := uuid.New()

	first, err := p.Publish(context.Background(), chID, publish.PostVariant{Text: "once", IdempotencyKey: "dupe-key"})
	if err != nil {
		t.Fatalf("first publish: %v", err)
	}
	second, err := p.Publish(context.Background(), chID, publish.PostVariant{Text: "once", IdempotencyKey: "dupe-key"})
	if err != nil {
		t.Fatalf("second publish: %v", err)
	}
	if second.PlatformPostID != first.PlatformPostID {
		t.Error("idempotent re-publish returned a different id")
	}
	if sim.TweetCount() != 1 {
		t.Errorf("tweet count = %d, want 1 (no double-post)", sim.TweetCount())
	}
}

func TestPipeline_RateLimit_RetriesThenSucceeds(t *testing.T) {
	a, sim, ch, res := harness(t)
	p := newPipeline(ch, res, a)
	sim.ForceNextCreateStatus(http.StatusTooManyRequests) // first attempt 429, then success

	out, err := p.Publish(context.Background(), uuid.New(), publish.PostVariant{Text: "retry me", IdempotencyKey: "k2"})
	if err != nil {
		t.Fatalf("Publish should succeed after backoff: %v", err)
	}
	if out.PlatformPostID == "" || sim.TweetCount() != 1 {
		t.Error("expected success after one 429 retry")
	}
}

func TestPipeline_ServerError_RetriesThenSucceeds(t *testing.T) {
	a, sim, ch, res := harness(t)
	p := newPipeline(ch, res, a)
	sim.ForceNextCreateStatus(http.StatusInternalServerError)

	if _, err := p.Publish(context.Background(), uuid.New(), publish.PostVariant{Text: "5xx then ok", IdempotencyKey: "k3"}); err != nil {
		t.Fatalf("Publish should succeed after 5xx retry: %v", err)
	}
}

func TestPipeline_ExpiredToken_RefreshesThenSucceeds(t *testing.T) {
	a, sim, ch, res := harness(t)
	p := newPipeline(ch, res, a)
	sim.ExpireAccessTokens() // current token invalid; refresh issues a fresh one

	out, err := p.Publish(context.Background(), uuid.New(), publish.PostVariant{Text: "needs refresh", IdempotencyKey: "k4"})
	if err != nil {
		t.Fatalf("Publish should succeed after refresh: %v", err)
	}
	if out.PlatformPostID == "" {
		t.Error("expected success after token refresh")
	}
	if ch.refreshCount != 1 {
		t.Errorf("refresh count = %d, want 1", ch.refreshCount)
	}
}

func TestPipeline_ExpiredToken_RefreshDoesNotConsumeAttempt(t *testing.T) {
	a, sim, ch, res := harness(t)
	// maxAttempts=1: the refresh must NOT burn the only attempt — the retry with
	// the refreshed token must still happen (regression guard).
	p := publish.NewPipeline(ch, res, []publish.Adapter{a},
		publish.WithSleeper(noSleep), publish.WithMaxAttempts(1))
	sim.ExpireAccessTokens()

	if _, err := p.Publish(context.Background(), uuid.New(), publish.PostVariant{Text: "one attempt", IdempotencyKey: "k-att1"}); err != nil {
		t.Fatalf("with maxAttempts=1, expired token should still publish after refresh: %v", err)
	}
	if ch.refreshCount != 1 {
		t.Errorf("refresh count = %d, want 1", ch.refreshCount)
	}
}

func TestPipeline_Duplicate_Terminal(t *testing.T) {
	a, _, ch, res := harness(t)
	p := newPipeline(ch, res, a)
	ctx := context.Background()

	if _, err := p.Publish(ctx, uuid.New(), publish.PostVariant{Text: "dup content", IdempotencyKey: "k5a"}); err != nil {
		t.Fatalf("first: %v", err)
	}
	// Different idempotency key, same content -> platform rejects as duplicate (terminal).
	_, err := p.Publish(ctx, uuid.New(), publish.PostVariant{Text: "dup content", IdempotencyKey: "k5b"})
	if err == nil {
		t.Fatal("duplicate content should fail terminally")
	}
}

func TestPipeline_OverLimit_TerminalNoAPICall(t *testing.T) {
	a, sim, ch, res := harness(t)
	p := newPipeline(ch, res, a)

	_, err := p.Publish(context.Background(), uuid.New(), publish.PostVariant{Text: string(make([]byte, 0)) + repeat("a", 281)})
	if err == nil {
		t.Fatal("over-limit should fail at validation")
	}
	if sim.TweetCount() != 0 {
		t.Error("validation failure should not reach the API")
	}
}

func repeat(s string, n int) string {
	b := make([]byte, 0, n)
	for i := 0; i < n; i++ {
		b = append(b, s...)
	}
	return string(b)
}
