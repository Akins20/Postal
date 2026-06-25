// Package telegram implements publish.Adapter for Telegram via the Bot API.
// Telegram does not use OAuth: a user creates a bot with @BotFather, adds it to
// a channel/group as an admin, and connects by supplying the bot token + chat
// id. Those are carried in the channel Token (AccessToken=bot token,
// RefreshToken=chat id). Publishing sends to the chat via sendMessage/sendPhoto/
// sendVideo.
package telegram

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/Akins20/postal/internal/channel"
	"github.com/Akins20/postal/internal/publish"
)

const (
	platformTelegram  = "telegram"
	defaultAPIBaseURL = "https://api.telegram.org"

	maxMessageRunes = 4096
	maxCaptionRunes = 1024
	maxImageBytes   = 10 << 20 // Telegram photo-by-URL limit
	maxVideoBytes   = 50 << 20 // bot upload limit; larger needs a local server
)

// Config configures the Telegram adapter. APIBaseURL is overridable so dev and
// tests point at the simulator.
type Config struct {
	APIBaseURL string
	HTTPClient *http.Client
}

// Adapter implements publish.Adapter for Telegram (Bot API).
type Adapter struct {
	base string
	http *http.Client
}

// New builds a Telegram adapter, applying defaults.
func New(cfg Config) *Adapter {
	base := cfg.APIBaseURL
	if base == "" {
		base = defaultAPIBaseURL
	}
	client := cfg.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	return &Adapter{base: base, http: client}
}

// Platform implements channel.OAuthProvider.
func (a *Adapter) Platform() string { return platformTelegram }

// --- channel.OAuthProvider: Telegram is not an OAuth provider, so the OAuth
// surface is stubbed. Connection happens via ConnectManual instead. ---

// AuthURL returns empty: Telegram has no authorize dialog.
func (a *Adapter) AuthURL(_, _, _ string) string { return "" }

// ExchangeCode is unsupported for Telegram (no OAuth code flow).
func (a *Adapter) ExchangeCode(_ context.Context, _, _, _ string) (*channel.Token, error) {
	return nil, publish.Terminal("oauth_unsupported", "Telegram connects with a bot token, not OAuth", nil)
}

// RefreshToken is a no-op: bot tokens do not expire.
func (a *Adapter) RefreshToken(_ context.Context, refreshToken string) (*channel.Token, error) {
	return nil, publish.Terminal("refresh_unsupported", "Telegram bot tokens do not refresh", nil)
}

// Account is unused for Telegram (ConnectManual resolves identity directly).
func (a *Adapter) Account(_ context.Context, _ string) (*channel.Account, error) {
	return nil, publish.Terminal("account_unsupported", "Telegram resolves identity at connect time", nil)
}

// Revoke is a no-op for bot tokens.
func (a *Adapter) Revoke(_ context.Context, _ string) error { return nil }

// ConnectManual implements channel.ManualConnector: it validates the supplied
// bot token + chat id against the Bot API and returns the channel token (token
// = bot token, refresh = chat id) and the resolved chat identity.
func (a *Adapter) ConnectManual(ctx context.Context, creds map[string]string) (*channel.Token, *channel.Account, error) {
	botToken := strings.TrimSpace(creds["bot_token"])
	chatID := strings.TrimSpace(creds["chat_id"])
	if botToken == "" || chatID == "" {
		return nil, nil, publish.Terminal("missing_credentials", "a Telegram bot token and chat id are both required", nil)
	}

	// getMe confirms the bot token is valid.
	var me struct {
		Username string `json:"username"`
	}
	if err := a.call(ctx, botToken, "getMe", nil, &me); err != nil {
		return nil, nil, err
	}

	// getChat confirms the bot can see the chat and resolves its title/handle.
	var chat struct {
		ID       json.Number `json:"id"`
		Title    string      `json:"title"`
		Username string      `json:"username"`
		Type     string      `json:"type"`
	}
	form := url.Values{}
	form.Set("chat_id", chatID)
	if err := a.call(ctx, botToken, "getChat", form, &chat); err != nil {
		return nil, nil, err
	}

	handle := chat.Username
	if handle != "" {
		handle = "@" + handle
	} else if chat.Title != "" {
		handle = chat.Title
	} else {
		handle = chatID
	}
	display := chat.Title
	if display == "" {
		display = handle
	}

	token := &channel.Token{
		AccessToken:  botToken,
		RefreshToken: chatID,
		Scopes:       []string{"bot"},
		ExpiresAt:    time.Now().Add(100 * 365 * 24 * time.Hour), // bot tokens do not expire
	}
	return token, &channel.Account{ID: chatID, Handle: handle, DisplayName: display}, nil
}

// Constraints implements publish.Adapter. Telegram allows text-only posts.
func (a *Adapter) Constraints() publish.Constraints {
	return publish.Constraints{
		RequiresMedia:      false,
		MaxWeightedTextLen: maxMessageRunes,
		MaxImages:          1,
		MaxVideos:          1,
		MaxImageBytes:      maxImageBytes,
		MaxVideoBytes:      maxVideoBytes,
	}
}

// Validate implements publish.Adapter.
func (a *Adapter) Validate(v publish.PostVariant) error {
	hasMedia := len(v.Media) > 0
	limit := maxMessageRunes
	if hasMedia {
		limit = maxCaptionRunes
	}
	if utf8.RuneCountInString(v.Text) > limit {
		return publish.Terminal("text_too_long",
			fmt.Sprintf("Telegram %s are limited to %d characters", map[bool]string{true: "captions", false: "messages"}[hasMedia], limit), nil)
	}
	if strings.TrimSpace(v.Text) == "" && !hasMedia {
		return publish.Terminal("empty_post", "a Telegram post needs text or media", nil)
	}
	if len(v.Media) > 1 {
		return publish.Terminal("too_many_media", "Telegram posts support one image or video for now", nil)
	}
	return nil
}

// Publish implements publish.Adapter. It sends to the chat carried in the token.
func (a *Adapter) Publish(ctx context.Context, token channel.Token, v publish.PostVariant) (*publish.Result, error) {
	if err := a.Validate(v); err != nil {
		return nil, err
	}
	botToken, chatID := token.AccessToken, token.RefreshToken
	if botToken == "" || chatID == "" {
		return nil, publish.Terminal("not_connected", "this Telegram channel is missing its bot token or chat id", nil)
	}

	form := url.Values{}
	form.Set("chat_id", chatID)
	var method string
	switch {
	case len(v.Media) == 0:
		method = "sendMessage"
		form.Set("text", v.Text)
	case v.Media[0].Kind == publish.MediaVideo:
		if v.Media[0].URL == "" {
			return nil, publish.Retryable("media_url_unavailable", "no public media URL available for Telegram", nil)
		}
		method = "sendVideo"
		form.Set("video", v.Media[0].URL)
		form.Set("caption", v.Text)
	default:
		if v.Media[0].URL == "" {
			return nil, publish.Retryable("media_url_unavailable", "no public media URL available for Telegram", nil)
		}
		method = "sendPhoto"
		form.Set("photo", v.Media[0].URL)
		form.Set("caption", v.Text)
	}

	var msg struct {
		MessageID json.Number `json:"message_id"`
	}
	if err := a.call(ctx, botToken, method, form, &msg); err != nil {
		return nil, err
	}
	return &publish.Result{PlatformPostID: fmt.Sprintf("%s:%s", chatID, msg.MessageID.String())}, nil
}

// FetchMetrics implements publish.Adapter. The Bot API exposes no per-message
// analytics, so this returns no metrics.
func (a *Adapter) FetchMetrics(_ context.Context, _ channel.Token, _ string) ([]publish.Metric, error) {
	return nil, nil
}

// call performs a Bot API method call (GET when form is nil, POST otherwise),
// unwrapping the {"ok":bool,"result":...,"description":...} envelope.
func (a *Adapter) call(ctx context.Context, botToken, method string, form url.Values, result any) error {
	endpoint := a.base + "/bot" + botToken + "/" + method
	var req *http.Request
	var err error
	if form == nil {
		req, err = http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	} else {
		req, err = http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
		if err == nil {
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		}
	}
	if err != nil {
		return fmt.Errorf("building telegram request: %w", err)
	}

	resp, err := a.http.Do(req)
	if err != nil {
		return publish.Retryable("network_error", "could not reach Telegram", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))

	var env struct {
		OK          bool            `json:"ok"`
		Result      json.RawMessage `json:"result"`
		Description string          `json:"description"`
		ErrorCode   int             `json:"error_code"`
		Parameters  struct {
			RetryAfter int `json:"retry_after"`
		} `json:"parameters"`
	}
	_ = json.Unmarshal(body, &env)
	if resp.StatusCode >= 200 && resp.StatusCode < 300 && env.OK {
		if result == nil {
			return nil
		}
		if err := json.Unmarshal(env.Result, result); err != nil {
			return publish.Terminal("bad_response", "could not parse Telegram response", err)
		}
		return nil
	}
	return classify(resp.StatusCode, env.ErrorCode, env.Description, env.Parameters.RetryAfter)
}

// classify maps Bot API errors onto adapter error classes.
func classify(status, errorCode int, desc string, retryAfter int) error {
	if desc == "" {
		desc = fmt.Sprintf("Telegram returned status %d", status)
	}
	switch {
	case status == http.StatusUnauthorized || errorCode == 401:
		return publish.AuthExpired(fmt.Errorf("telegram: %s", desc))
	case status == http.StatusTooManyRequests || errorCode == 429:
		return publish.RateLimited(time.Duration(retryAfter)*time.Second, fmt.Errorf("telegram: %s", desc))
	case status >= 500:
		return publish.Retryable("server_error", desc, nil)
	default:
		return publish.Terminal("telegram_error", desc, nil)
	}
}
