package twitter_test

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/Akins20/postal/internal/channel"
	"github.com/Akins20/postal/internal/publish"
	twittersim "github.com/Akins20/postal/internal/publish/simulator/twitter"
	"github.com/Akins20/postal/internal/publish/twitter"
)

// setup starts a simulator and an adapter pointed at it, returning an
// authenticated token.
func setup(t *testing.T) (*twitter.Adapter, *twittersim.Server, channel.Token) {
	t.Helper()
	sim := twittersim.New()
	t.Cleanup(sim.Close)

	a := twitter.New(twitter.Config{
		ClientID:    "test-client",
		RedirectURI: "https://app.test/cb",
		APIBaseURL:  sim.URL(),
		AuthBaseURL: sim.URL(),
	})
	tok, err := a.ExchangeCode(context.Background(), "auth-code", "verifier")
	if err != nil {
		t.Fatalf("ExchangeCode: %v", err)
	}
	if tok.AccessToken == "" || tok.RefreshToken == "" {
		t.Fatal("token exchange did not return tokens")
	}
	return a, sim, *tok
}

func wantClass(t *testing.T, err error, want publish.Class) {
	t.Helper()
	var e *publish.Error
	if !errors.As(err, &e) {
		t.Fatalf("error is not *publish.Error: %v", err)
	}
	if e.Class != want {
		t.Fatalf("error class = %d, want %d (%v)", e.Class, want, err)
	}
}

func TestAdapter_AuthURL(t *testing.T) {
	a := twitter.New(twitter.Config{ClientID: "cid", RedirectURI: "https://app/cb", AuthBaseURL: "https://x.com"})
	u := a.AuthURL("state123", "challenge456")
	for _, want := range []string{"/i/oauth2/authorize", "response_type=code", "code_challenge=challenge456", "code_challenge_method=S256", "state=state123", "scope="} {
		if !strings.Contains(u, want) {
			t.Errorf("AuthURL missing %q: %s", want, u)
		}
	}
}

func TestAdapter_AccountAndHappyPath(t *testing.T) {
	a, sim, tok := setup(t)
	ctx := context.Background()

	acct, err := a.Account(ctx, tok.AccessToken)
	if err != nil {
		t.Fatalf("Account: %v", err)
	}
	if acct.ID != "sim-user-1" || acct.Handle != "@simuser" {
		t.Errorf("unexpected account: %+v", acct)
	}

	res, err := a.Publish(ctx, tok, publish.PostVariant{Text: "hello world"})
	if err != nil {
		t.Fatalf("Publish: %v", err)
	}
	if res.PlatformPostID == "" {
		t.Error("no platform post id")
	}
	if sim.TweetCount() != 1 {
		t.Errorf("tweet count = %d, want 1", sim.TweetCount())
	}
}

func TestAdapter_PublishWithImage(t *testing.T) {
	a, sim, tok := setup(t)
	res, err := a.Publish(context.Background(), tok, publish.PostVariant{
		Text:  "with media",
		Media: []publish.MediaRef{{Kind: publish.MediaImage, MIME: "image/png", Bytes: 1024, Data: make([]byte, 1024)}},
	})
	if err != nil {
		t.Fatalf("Publish with image: %v", err)
	}
	if res.PlatformPostID == "" || sim.TweetCount() != 1 {
		t.Error("media publish did not create a tweet")
	}
}

func TestAdapter_PublishVideoExercisesStatusPoll(t *testing.T) {
	a, sim, tok := setup(t)
	sim.EnableMediaProcessing() // FINALIZE -> processing; STATUS -> succeeded

	res, err := a.Publish(context.Background(), tok, publish.PostVariant{
		Text:  "a video post",
		Media: []publish.MediaRef{{Kind: publish.MediaVideo, MIME: "video/mp4", Bytes: 2048, Data: make([]byte, 2048)}},
	})
	if err != nil {
		t.Fatalf("Publish video (async processing): %v", err)
	}
	if res.PlatformPostID == "" || sim.TweetCount() != 1 {
		t.Error("video publish via STATUS-poll did not create a tweet")
	}
}

func TestAdapter_OverLimitRejectedAtValidate(t *testing.T) {
	a, sim, tok := setup(t)
	long := strings.Repeat("a", 281)

	_, err := a.Publish(context.Background(), tok, publish.PostVariant{Text: long})
	wantClass(t, err, publish.ClassTerminal)
	if sim.TweetCount() != 0 {
		t.Error("over-limit post should not reach the API")
	}
}

func TestAdapter_DuplicateTerminal(t *testing.T) {
	a, _, tok := setup(t)
	ctx := context.Background()
	if _, err := a.Publish(ctx, tok, publish.PostVariant{Text: "same text"}); err != nil {
		t.Fatalf("first publish: %v", err)
	}
	_, err := a.Publish(ctx, tok, publish.PostVariant{Text: "same text"})
	wantClass(t, err, publish.ClassTerminal)
}

func TestAdapter_RateLimitRetryable(t *testing.T) {
	a, sim, tok := setup(t)
	sim.ForceNextCreateStatus(http.StatusTooManyRequests)

	_, err := a.Publish(context.Background(), tok, publish.PostVariant{Text: "rate me"})
	wantClass(t, err, publish.ClassRetryable)
	var e *publish.Error
	errors.As(err, &e)
	if e.RetryAfter <= 0 {
		t.Errorf("429 should carry RetryAfter, got %v", e.RetryAfter)
	}
}

func TestAdapter_ServerErrorRetryable(t *testing.T) {
	a, sim, tok := setup(t)
	sim.ForceNextCreateStatus(http.StatusInternalServerError)
	_, err := a.Publish(context.Background(), tok, publish.PostVariant{Text: "boom"})
	wantClass(t, err, publish.ClassRetryable)
}

func TestAdapter_ExpiredTokenAuthExpired(t *testing.T) {
	a, sim, tok := setup(t)
	sim.ExpireAccessTokens()
	_, err := a.Publish(context.Background(), tok, publish.PostVariant{Text: "after expiry"})
	wantClass(t, err, publish.ClassAuthExpired)
}

func TestAdapter_RefreshTokenAfterExpiry(t *testing.T) {
	a, sim, tok := setup(t)
	ctx := context.Background()
	sim.ExpireAccessTokens()

	newTok, err := a.RefreshToken(ctx, tok.RefreshToken)
	if err != nil {
		t.Fatalf("RefreshToken: %v", err)
	}
	// The refreshed access token works again.
	if _, err := a.Publish(ctx, *newTok, publish.PostVariant{Text: "fresh"}); err != nil {
		t.Fatalf("publish after refresh: %v", err)
	}
}

func TestAdapter_FetchMetrics(t *testing.T) {
	a, _, tok := setup(t)
	ctx := context.Background()
	res, err := a.Publish(ctx, tok, publish.PostVariant{Text: "measure me"})
	if err != nil {
		t.Fatalf("Publish: %v", err)
	}
	metrics, err := a.FetchMetrics(ctx, tok, res.PlatformPostID)
	if err != nil {
		t.Fatalf("FetchMetrics: %v", err)
	}
	got := map[string]int64{}
	for _, m := range metrics {
		got[m.Name] = m.Value
	}
	if got["likes"] != 5 || got["impressions"] != 100 {
		t.Errorf("unexpected metrics: %+v", got)
	}
}
