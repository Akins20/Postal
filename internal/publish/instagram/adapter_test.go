package instagram_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Akins20/postal/internal/channel"
	"github.com/Akins20/postal/internal/publish"
	"github.com/Akins20/postal/internal/publish/instagram"
	instagramsim "github.com/Akins20/postal/internal/publish/simulator/instagram"
)

func newAdapter(t *testing.T) (*instagram.Adapter, *instagramsim.Server) {
	t.Helper()
	sim := instagramsim.New()
	t.Cleanup(sim.Close)
	return instagram.New(instagram.Config{
		ClientID: "ig-app", ClientSecret: "ig-secret",
		RedirectURI: "http://localhost:3000/oauth/callback",
		APIBaseURL:  sim.URL(), AuthBaseURL: sim.URL(),
	}), sim
}

func connect(t *testing.T, a *instagram.Adapter) channel.Token {
	t.Helper()
	tok, err := a.ExchangeCode(context.Background(), "igcode-test", "")
	require.NoError(t, err)
	return *tok
}

func TestOAuthAndIdentity(t *testing.T) {
	a, _ := newAdapter(t)
	auth := a.AuthURL("state-1", "challenge")
	assert.Contains(t, auth, "/v21.0/dialog/oauth?")
	assert.Contains(t, auth, "state=state-1")

	tok := connect(t, a)
	assert.NotEmpty(t, tok.AccessToken)
	assert.Equal(t, tok.AccessToken, tok.RefreshToken, "long-lived token doubles as refresh material")

	acct, err := a.Account(context.Background(), tok.AccessToken)
	require.NoError(t, err)
	assert.Equal(t, "ig-1", acct.ID)
	assert.Equal(t, "@simgram", acct.Handle)

	refreshed, err := a.RefreshToken(context.Background(), tok.RefreshToken)
	require.NoError(t, err)
	assert.NotEmpty(t, refreshed.AccessToken)
}

func TestValidateRejectsTextOnlyAndOversize(t *testing.T) {
	a, _ := newAdapter(t)
	assert.Error(t, a.Validate(publish.PostVariant{Text: "no media"}), "text-only must fail")
	long := strings.Repeat("a", 2201)
	assert.Error(t, a.Validate(publish.PostVariant{
		Text:  long,
		Media: []publish.MediaRef{{Kind: publish.MediaImage, Bytes: 10}},
	}))
	assert.Error(t, a.Validate(publish.PostVariant{
		Media: []publish.MediaRef{{Kind: publish.MediaGIF, Bytes: 10}},
	}), "GIFs unsupported")
	assert.NoError(t, a.Validate(publish.PostVariant{
		Text:  "ok",
		Media: []publish.MediaRef{{Kind: publish.MediaImage, Bytes: 10}},
	}))
}

func TestPublishContainerFlow(t *testing.T) {
	a, sim := newAdapter(t)
	tok := connect(t, a)

	res, err := a.Publish(context.Background(), tok, publish.PostVariant{
		Text: "hello insta",
		Media: []publish.MediaRef{{
			Kind: publish.MediaImage, MIME: "image/png", Bytes: 100,
			URL: "https://media.test/presigned.png",
		}},
	})
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(res.PlatformPostID, "igpost-"))
	assert.Equal(t, 1, sim.PostCount())

	metrics, err := a.FetchMetrics(context.Background(), tok, res.PlatformPostID)
	require.NoError(t, err)
	require.NotEmpty(t, metrics)
	assert.Equal(t, "likes", metrics[0].Name)
}

func TestPublishWaitsOutProcessingContainers(t *testing.T) {
	instagram.SetContainerPollWaitForTest(t, 10*time.Millisecond)
	a, sim := newAdapter(t)
	tok := connect(t, a)
	sim.EnableProcessing(2) // two IN_PROGRESS polls before FINISHED

	res, err := a.Publish(context.Background(), tok, publish.PostVariant{
		Text: "video time",
		Media: []publish.MediaRef{{
			Kind: publish.MediaVideo, MIME: "video/mp4", Bytes: 1000,
			URL: "https://media.test/presigned.mp4",
		}},
	})
	require.NoError(t, err)
	assert.NotEmpty(t, res.PlatformPostID)
}

func TestPublishWithoutPresignedURLIsRetryable(t *testing.T) {
	a, _ := newAdapter(t)
	tok := connect(t, a)
	_, err := a.Publish(context.Background(), tok, publish.PostVariant{
		Text:  "no url",
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
