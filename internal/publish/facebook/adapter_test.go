package facebook_test

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Akins20/postal/internal/channel"
	"github.com/Akins20/postal/internal/publish"
	"github.com/Akins20/postal/internal/publish/facebook"
	facebooksim "github.com/Akins20/postal/internal/publish/simulator/facebook"
)

func newAdapter(t *testing.T) (*facebook.Adapter, *facebooksim.Server) {
	t.Helper()
	sim := facebooksim.New()
	t.Cleanup(sim.Close)
	return facebook.New(facebook.Config{
		ClientID: "fb-app", ClientSecret: "fb-secret",
		RedirectURI: "http://localhost:3000/oauth/callback",
		APIBaseURL:  sim.URL(), AuthBaseURL: sim.URL(),
	}), sim
}

func connect(t *testing.T, a *facebook.Adapter) channel.Token {
	t.Helper()
	tok, err := a.ExchangeCode(context.Background(), "fbcode-test", "", "")
	require.NoError(t, err)
	return *tok
}

func TestOAuthAndIdentity(t *testing.T) {
	a, _ := newAdapter(t)
	auth := a.AuthURL("state-1", "challenge", "")
	assert.Contains(t, auth, "/v21.0/dialog/oauth?")
	assert.Contains(t, auth, "state=state-1")
	assert.Contains(t, auth, "pages_manage_posts")

	tok := connect(t, a)
	assert.NotEmpty(t, tok.AccessToken)
	assert.Equal(t, tok.AccessToken, tok.RefreshToken)

	acct, err := a.Account(context.Background(), tok.AccessToken)
	require.NoError(t, err)
	assert.Equal(t, "fbpage-1", acct.ID)
	assert.Equal(t, "Sim Page", acct.DisplayName)

	refreshed, err := a.RefreshToken(context.Background(), tok.RefreshToken)
	require.NoError(t, err)
	assert.NotEmpty(t, refreshed.AccessToken)
}

func TestValidate(t *testing.T) {
	a, _ := newAdapter(t)
	// Text-only is allowed on Facebook.
	assert.NoError(t, a.Validate(publish.PostVariant{Text: "just text"}))
	// Empty post is rejected.
	assert.Error(t, a.Validate(publish.PostVariant{}))
	// Over-length message is rejected.
	assert.Error(t, a.Validate(publish.PostVariant{Text: strings.Repeat("a", 63207)}))
	// More than one media is rejected.
	assert.Error(t, a.Validate(publish.PostVariant{
		Media: []publish.MediaRef{{Kind: publish.MediaImage, Bytes: 10}, {Kind: publish.MediaImage, Bytes: 10}},
	}))
}

func TestPublishTextOnly(t *testing.T) {
	a, sim := newAdapter(t)
	tok := connect(t, a)
	res, err := a.Publish(context.Background(), tok, publish.PostVariant{Text: "hello facebook https://postal.app"})
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(res.PlatformPostID, "fbpage-1_"))
	assert.Equal(t, 1, sim.PostCount())

	metrics, err := a.FetchMetrics(context.Background(), tok, res.PlatformPostID)
	require.NoError(t, err)
	require.NotEmpty(t, metrics)
	assert.Equal(t, "post_impressions", metrics[0].Name)
}

func TestPublishPhotoAndVideo(t *testing.T) {
	a, sim := newAdapter(t)
	tok := connect(t, a)

	_, err := a.Publish(context.Background(), tok, publish.PostVariant{
		Text:  "a photo",
		Media: []publish.MediaRef{{Kind: publish.MediaImage, Bytes: 100, URL: "https://media.test/p.png"}},
	})
	require.NoError(t, err)

	_, err = a.Publish(context.Background(), tok, publish.PostVariant{
		Text:  "a video",
		Media: []publish.MediaRef{{Kind: publish.MediaVideo, Bytes: 1000, URL: "https://media.test/v.mp4"}},
	})
	require.NoError(t, err)
	assert.Equal(t, 2, sim.PostCount())
}

func TestPublishMediaWithoutPresignedURLIsRetryable(t *testing.T) {
	a, _ := newAdapter(t)
	tok := connect(t, a)
	_, err := a.Publish(context.Background(), tok, publish.PostVariant{
		Media: []publish.MediaRef{{Kind: publish.MediaImage, Bytes: 10}},
	})
	var pe *publish.Error
	require.ErrorAs(t, err, &pe)
	assert.Equal(t, publish.ClassRetryable, pe.Class)
}

func TestExpiredTokenMapsToAuthExpired(t *testing.T) {
	a, _ := newAdapter(t)
	_, err := a.Account(context.Background(), "bogus-token")
	var pe *publish.Error
	require.ErrorAs(t, err, &pe)
	assert.Equal(t, publish.ClassAuthExpired, pe.Class)
}
