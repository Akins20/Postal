package tiktok_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Akins20/postal/internal/channel"
	"github.com/Akins20/postal/internal/publish"
	tiktoksim "github.com/Akins20/postal/internal/publish/simulator/tiktok"
	"github.com/Akins20/postal/internal/publish/tiktok"
)

func newAdapter(t *testing.T) (*tiktok.Adapter, *tiktoksim.Server) {
	t.Helper()
	sim := tiktoksim.New()
	t.Cleanup(sim.Close)
	return tiktok.New(tiktok.Config{
		ClientKey: "tt-key", ClientSecret: "tt-secret",
		RedirectURI: "http://localhost:3000/oauth/callback",
		APIBaseURL:  sim.URL(), AuthBaseURL: sim.URL(),
	}), sim
}

func connect(t *testing.T, a *tiktok.Adapter) channel.Token {
	t.Helper()
	tok, err := a.ExchangeCode(context.Background(), "ttcode-test", "verifier")
	require.NoError(t, err)
	return *tok
}

func TestOAuthAndIdentity(t *testing.T) {
	a, _ := newAdapter(t)
	auth := a.AuthURL("state-1", "challenge")
	assert.Contains(t, auth, "/v2/auth/authorize/?")
	assert.Contains(t, auth, "client_key=tt-key")
	assert.Contains(t, auth, "code_challenge=challenge")

	tok := connect(t, a)
	assert.NotEmpty(t, tok.AccessToken)
	assert.NotEmpty(t, tok.RefreshToken)

	acct, err := a.Account(context.Background(), tok.AccessToken)
	require.NoError(t, err)
	assert.Equal(t, "tt-open-1", acct.ID)
	assert.Equal(t, "@simtok", acct.Handle)

	refreshed, err := a.RefreshToken(context.Background(), tok.RefreshToken)
	require.NoError(t, err)
	assert.NotEmpty(t, refreshed.AccessToken)
}

func TestValidateShapes(t *testing.T) {
	a, _ := newAdapter(t)
	assert.Error(t, a.Validate(publish.PostVariant{Text: "text only"}), "text-only must fail")
	assert.Error(t, a.Validate(publish.PostVariant{Media: []publish.MediaRef{
		{Kind: publish.MediaVideo, Bytes: 10}, {Kind: publish.MediaImage, Bytes: 10},
	}}), "video+photos must fail")
	assert.Error(t, a.Validate(publish.PostVariant{Media: []publish.MediaRef{
		{Kind: publish.MediaGIF, Bytes: 10},
	}}), "GIFs unsupported")
	assert.NoError(t, a.Validate(publish.PostVariant{
		Text:  "ok",
		Media: []publish.MediaRef{{Kind: publish.MediaVideo, Bytes: 100}},
	}))
}

func TestPublishVideoUploadsBytes(t *testing.T) {
	tiktok.SetStatusPollWaitForTest(t, 10*time.Millisecond)
	a, sim := newAdapter(t)
	tok := connect(t, a)

	data := []byte(strings.Repeat("v", 4096))
	res, err := a.Publish(context.Background(), tok, publish.PostVariant{
		Text:  "tok tok",
		Media: []publish.MediaRef{{Kind: publish.MediaVideo, MIME: "video/mp4", Bytes: int64(len(data)), Data: data}},
	})
	require.NoError(t, err)
	assert.NotEmpty(t, res.PlatformPostID)
	assert.Equal(t, 1, sim.PostCount())

	metrics, err := a.FetchMetrics(context.Background(), tok, res.PlatformPostID)
	require.NoError(t, err)
	require.Len(t, metrics, 4)
	assert.Equal(t, "views", metrics[0].Name)
	assert.Equal(t, int64(240), metrics[0].Value)
}

func TestPublishPhotosViaPresignedURLs(t *testing.T) {
	tiktok.SetStatusPollWaitForTest(t, 10*time.Millisecond)
	a, sim := newAdapter(t)
	tok := connect(t, a)

	res, err := a.Publish(context.Background(), tok, publish.PostVariant{
		Text: "photo dump",
		Media: []publish.MediaRef{
			{Kind: publish.MediaImage, Bytes: 10, URL: "https://media.test/a.png"},
			{Kind: publish.MediaImage, Bytes: 10, URL: "https://media.test/b.png"},
		},
	})
	require.NoError(t, err)
	assert.NotEmpty(t, res.PlatformPostID)
	assert.Equal(t, 1, sim.PostCount())
}

func TestPublishSurvivesProcessingDelay(t *testing.T) {
	tiktok.SetStatusPollWaitForTest(t, 10*time.Millisecond)
	a, sim := newAdapter(t)
	tok := connect(t, a)
	sim.EnableProcessing(3)

	data := []byte("tiny video")
	res, err := a.Publish(context.Background(), tok, publish.PostVariant{
		Text:  "slow cook",
		Media: []publish.MediaRef{{Kind: publish.MediaVideo, MIME: "video/mp4", Bytes: int64(len(data)), Data: data}},
	})
	require.NoError(t, err)
	assert.NotEmpty(t, res.PlatformPostID)
}

func TestUnauditedAppStillPublishesPrivately(t *testing.T) {
	tiktok.SetStatusPollWaitForTest(t, 10*time.Millisecond)
	a, sim := newAdapter(t)
	tok := connect(t, a)
	sim.SetSelfOnly(true) // unaudited API client: SELF_ONLY is the only option

	data := []byte("private video")
	res, err := a.Publish(context.Background(), tok, publish.PostVariant{
		Text:  "audit pending",
		Media: []publish.MediaRef{{Kind: publish.MediaVideo, MIME: "video/mp4", Bytes: int64(len(data)), Data: data}},
	})
	require.NoError(t, err)
	assert.NotEmpty(t, res.PlatformPostID)
}

func TestExpiredTokenMapsToAuthExpired(t *testing.T) {
	a, _ := newAdapter(t)
	_, err := a.Account(context.Background(), "bogus")
	var pe *publish.Error
	require.ErrorAs(t, err, &pe)
	assert.Equal(t, publish.ClassAuthExpired, pe.Class)
}
