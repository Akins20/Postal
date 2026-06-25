package telegram_test

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Akins20/postal/internal/channel"
	"github.com/Akins20/postal/internal/publish"
	"github.com/Akins20/postal/internal/publish/telegram"
	telegramsim "github.com/Akins20/postal/internal/publish/simulator/telegram"
)

func newAdapter(t *testing.T) (*telegram.Adapter, *telegramsim.Server) {
	t.Helper()
	sim := telegramsim.New()
	t.Cleanup(sim.Close)
	return telegram.New(telegram.Config{APIBaseURL: sim.URL()}), sim
}

func TestConnectManual(t *testing.T) {
	a, _ := newAdapter(t)

	// Missing credentials.
	_, _, err := a.ConnectManual(context.Background(), map[string]string{"bot_token": telegramsim.ValidToken})
	require.Error(t, err)

	// Invalid bot token -> auth error.
	_, _, err = a.ConnectManual(context.Background(), map[string]string{"bot_token": "bad", "chat_id": "@x"})
	var pe *publish.Error
	require.ErrorAs(t, err, &pe)
	assert.Equal(t, publish.ClassAuthExpired, pe.Class)

	// Valid -> token carries bot token + chat id, account resolved.
	tok, acct, err := a.ConnectManual(context.Background(),
		map[string]string{"bot_token": telegramsim.ValidToken, "chat_id": "@simchannel"})
	require.NoError(t, err)
	assert.Equal(t, telegramsim.ValidToken, tok.AccessToken)
	assert.Equal(t, "@simchannel", tok.RefreshToken)
	assert.Equal(t, "@simchannel", acct.Handle)
	assert.Equal(t, "@simchannel", acct.ID)
}

func TestValidate(t *testing.T) {
	a, _ := newAdapter(t)
	assert.NoError(t, a.Validate(publish.PostVariant{Text: "just text"}))
	assert.Error(t, a.Validate(publish.PostVariant{}))
	assert.Error(t, a.Validate(publish.PostVariant{Text: strings.Repeat("a", 4097)}))
	// Caption limit is tighter when media is attached.
	assert.Error(t, a.Validate(publish.PostVariant{
		Text:  strings.Repeat("a", 1025),
		Media: []publish.MediaRef{{Kind: publish.MediaImage, Bytes: 10}},
	}))
}

func connectTok() channel.Token {
	return channel.Token{AccessToken: telegramsim.ValidToken, RefreshToken: "@simchannel"}
}

func TestPublishTextPhotoVideo(t *testing.T) {
	a, sim := newAdapter(t)
	tok := connectTok()

	res, err := a.Publish(context.Background(), tok, publish.PostVariant{Text: "hello telegram"})
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(res.PlatformPostID, "@simchannel:"))

	_, err = a.Publish(context.Background(), tok, publish.PostVariant{
		Text:  "photo",
		Media: []publish.MediaRef{{Kind: publish.MediaImage, Bytes: 100, URL: "https://media.test/p.png"}},
	})
	require.NoError(t, err)

	_, err = a.Publish(context.Background(), tok, publish.PostVariant{
		Text:  "video",
		Media: []publish.MediaRef{{Kind: publish.MediaVideo, Bytes: 100, URL: "https://media.test/v.mp4"}},
	})
	require.NoError(t, err)
	assert.Equal(t, 3, sim.PostCount())
}

func TestPublishMediaWithoutURLIsRetryable(t *testing.T) {
	a, _ := newAdapter(t)
	_, err := a.Publish(context.Background(), connectTok(), publish.PostVariant{
		Media: []publish.MediaRef{{Kind: publish.MediaImage, Bytes: 10}},
	})
	var pe *publish.Error
	require.ErrorAs(t, err, &pe)
	assert.Equal(t, publish.ClassRetryable, pe.Class)
}

func TestPublishWithBadTokenIsAuthExpired(t *testing.T) {
	a, _ := newAdapter(t)
	_, err := a.Publish(context.Background(), channel.Token{AccessToken: "bad", RefreshToken: "@x"},
		publish.PostVariant{Text: "x"})
	var pe *publish.Error
	require.ErrorAs(t, err, &pe)
	assert.Equal(t, publish.ClassAuthExpired, pe.Class)
}

func TestOAuthSurfaceIsStubbed(t *testing.T) {
	a, _ := newAdapter(t)
	assert.Equal(t, "", a.AuthURL("s", "c", ""))
	_, err := a.ExchangeCode(context.Background(), "code", "", "")
	require.Error(t, err)
}
