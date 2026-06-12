// Package instagram implements publish.Adapter for Instagram via the Meta
// Graph API. Publishing is the documented container flow: create a media
// container from a PUBLIC media URL (MediaRef.URL, presigned by storage),
// poll its status, then publish it. Business/Creator accounts only; no
// text-only posts. See docs/PLATFORMS_IG_TIKTOK.md.
package instagram

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
	platformInstagram  = "instagram"
	defaultAPIBaseURL  = "https://graph.facebook.com"
	defaultAuthBaseURL = "https://www.facebook.com"
	apiVersion         = "v21.0"

	maxCaptionRunes = 2200
	maxImageBytes   = 8 << 20 // 8 MiB images
	maxVideoBytes   = 1 << 30 // 1 GiB reels
	oauthScopes     = "instagram_basic,instagram_content_publish,pages_show_list,business_management"

	// containerPolls bounds status checks per publish attempt; a container
	// still processing afterwards returns retryable so asynq re-runs the job
	// later instead of blocking a worker.
	containerPolls = 10
)

// containerPollWait spaces container status checks. A var so tests can lower
// it without waiting out real processing windows.
var containerPollWait = 3 * time.Second

// Config configures the Instagram adapter. Base URLs are overridable so dev
// and tests point at the simulator.
type Config struct {
	ClientID     string
	ClientSecret string
	RedirectURI  string
	APIBaseURL   string
	AuthBaseURL  string
	HTTPClient   *http.Client
}

// Adapter implements publish.Adapter for Instagram (Meta Graph API).
type Adapter struct {
	cfg  Config
	http *http.Client
}

// New builds an Instagram adapter, applying default hosts and client.
func New(cfg Config) *Adapter {
	if cfg.APIBaseURL == "" {
		cfg.APIBaseURL = defaultAPIBaseURL
	}
	if cfg.AuthBaseURL == "" {
		cfg.AuthBaseURL = defaultAuthBaseURL
	}
	if cfg.HTTPClient == nil {
		cfg.HTTPClient = &http.Client{Timeout: 30 * time.Second}
	}
	return &Adapter{cfg: cfg, http: cfg.HTTPClient}
}

// Platform implements channel.OAuthProvider.
func (a *Adapter) Platform() string { return platformInstagram }

// Constraints implements publish.Adapter.
func (a *Adapter) Constraints() publish.Constraints {
	return publish.Constraints{
		RequiresMedia:      true,
		MaxWeightedTextLen: maxCaptionRunes,
		MaxImages:          1,
		MaxVideos:          1,
		MaxImageBytes:      maxImageBytes,
		MaxVideoBytes:      maxVideoBytes,
	}
}

// Validate implements publish.Adapter: captions are capped and a post must
// carry exactly one image or video (carousels are a later enhancement).
func (a *Adapter) Validate(v publish.PostVariant) error {
	if utf8.RuneCountInString(v.Text) > maxCaptionRunes {
		return publish.Terminal("caption_too_long",
			fmt.Sprintf("Instagram captions are limited to %d characters", maxCaptionRunes), nil)
	}
	if len(v.Media) == 0 {
		return publish.Terminal("media_required",
			"Instagram does not allow text-only posts; attach an image or video", nil)
	}
	if len(v.Media) > 1 {
		return publish.Terminal("too_many_media",
			"Instagram posts support one image or video for now", nil)
	}
	m := v.Media[0]
	if m.Kind == publish.MediaGIF {
		return publish.Terminal("unsupported_media", "Instagram does not accept GIF uploads", nil)
	}
	if m.Kind == publish.MediaImage && m.Bytes > maxImageBytes {
		return publish.Terminal("image_too_large", "Instagram images are limited to 8 MiB", nil)
	}
	if m.Kind == publish.MediaVideo && m.Bytes > maxVideoBytes {
		return publish.Terminal("video_too_large", "Instagram videos are limited to 1 GiB", nil)
	}
	return nil
}

// AuthURL implements channel.OAuthProvider. Meta's dialog has no PKCE; the
// challenge parameter is accepted and ignored.
func (a *Adapter) AuthURL(state, _ string) string {
	v := url.Values{}
	v.Set("client_id", a.cfg.ClientID)
	v.Set("redirect_uri", a.cfg.RedirectURI)
	v.Set("state", state)
	v.Set("scope", oauthScopes)
	v.Set("response_type", "code")
	return a.cfg.AuthBaseURL + "/" + apiVersion + "/dialog/oauth?" + v.Encode()
}

// ExchangeCode implements channel.OAuthProvider: code -> short-lived token ->
// long-lived (~60 day) token. Meta has no refresh grant; the long-lived token
// itself is stored as the "refresh token" and re-exchanged before expiry.
func (a *Adapter) ExchangeCode(ctx context.Context, code, _ string) (*channel.Token, error) {
	v := url.Values{}
	v.Set("client_id", a.cfg.ClientID)
	v.Set("client_secret", a.cfg.ClientSecret)
	v.Set("redirect_uri", a.cfg.RedirectURI)
	v.Set("code", code)
	var short struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int64  `json:"expires_in"`
	}
	if err := a.getJSON(ctx, "/oauth/access_token?"+v.Encode(), "", &short); err != nil {
		return nil, err
	}
	return a.longLived(ctx, short.AccessToken)
}

// RefreshToken implements channel.OAuthProvider by re-exchanging the stored
// long-lived token for a fresh one.
func (a *Adapter) RefreshToken(ctx context.Context, refreshToken string) (*channel.Token, error) {
	return a.longLived(ctx, refreshToken)
}

func (a *Adapter) longLived(ctx context.Context, token string) (*channel.Token, error) {
	v := url.Values{}
	v.Set("grant_type", "fb_exchange_token")
	v.Set("client_id", a.cfg.ClientID)
	v.Set("client_secret", a.cfg.ClientSecret)
	v.Set("fb_exchange_token", token)
	var out struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int64  `json:"expires_in"`
	}
	if err := a.getJSON(ctx, "/oauth/access_token?"+v.Encode(), "", &out); err != nil {
		return nil, err
	}
	expires := time.Now().Add(60 * 24 * time.Hour)
	if out.ExpiresIn > 0 {
		expires = time.Now().Add(time.Duration(out.ExpiresIn) * time.Second)
	}
	return &channel.Token{
		AccessToken:  out.AccessToken,
		RefreshToken: out.AccessToken, // re-exchanged on refresh
		Scopes:       strings.Split(oauthScopes, ","),
		ExpiresAt:    expires,
	}, nil
}

// Account implements channel.OAuthProvider with Meta's multi-hop resolution:
// pages -> page's instagram_business_account -> IG username.
func (a *Adapter) Account(ctx context.Context, accessToken string) (*channel.Account, error) {
	igID, err := a.igUserID(ctx, accessToken)
	if err != nil {
		return nil, err
	}
	var ig struct {
		ID       string `json:"id"`
		Username string `json:"username"`
		Name     string `json:"name"`
	}
	if err := a.getJSON(ctx, "/"+igID+"?fields=username,name", accessToken, &ig); err != nil {
		return nil, err
	}
	handle := ig.Username
	if handle != "" {
		handle = "@" + handle
	}
	return &channel.Account{ID: ig.ID, Handle: handle, DisplayName: ig.Name}, nil
}

// igUserID resolves the Instagram Business account behind the user's pages.
func (a *Adapter) igUserID(ctx context.Context, accessToken string) (string, error) {
	var pages struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := a.getJSON(ctx, "/me/accounts", accessToken, &pages); err != nil {
		return "", err
	}
	for _, page := range pages.Data {
		var pg struct {
			InstagramBusinessAccount *struct {
				ID string `json:"id"`
			} `json:"instagram_business_account"`
		}
		if err := a.getJSON(ctx, "/"+page.ID+"?fields=instagram_business_account", accessToken, &pg); err != nil {
			return "", err
		}
		if pg.InstagramBusinessAccount != nil && pg.InstagramBusinessAccount.ID != "" {
			return pg.InstagramBusinessAccount.ID, nil
		}
	}
	return "", publish.Terminal("no_instagram_business_account",
		"no Instagram Business/Creator account is linked to your Facebook pages", nil)
}

// Revoke implements channel.OAuthProvider (best effort).
func (a *Adapter) Revoke(ctx context.Context, token string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete,
		a.cfg.APIBaseURL+"/"+apiVersion+"/me/permissions?access_token="+url.QueryEscape(token), nil)
	if err != nil {
		return fmt.Errorf("building revoke request: %w", err)
	}
	resp, err := a.http.Do(req)
	if err != nil {
		return fmt.Errorf("revoking instagram token: %w", err)
	}
	_ = resp.Body.Close()
	return nil
}

// Publish implements publish.Adapter via the container flow.
func (a *Adapter) Publish(ctx context.Context, token channel.Token, v publish.PostVariant) (*publish.Result, error) {
	if err := a.Validate(v); err != nil {
		return nil, err
	}
	m := v.Media[0]
	if m.URL == "" {
		return nil, publish.Retryable("media_url_unavailable",
			"no public media URL available for Instagram (storage presigning failed)", nil)
	}
	igID, err := a.igUserID(ctx, token.AccessToken)
	if err != nil {
		return nil, err
	}

	form := url.Values{}
	form.Set("caption", v.Text)
	if m.Kind == publish.MediaVideo {
		form.Set("media_type", "REELS")
		form.Set("video_url", m.URL)
	} else {
		form.Set("image_url", m.URL)
	}
	var container struct {
		ID string `json:"id"`
	}
	if err := a.postForm(ctx, "/"+igID+"/media", token.AccessToken, form, &container); err != nil {
		return nil, err
	}

	if err := a.waitForContainer(ctx, token.AccessToken, container.ID); err != nil {
		return nil, err
	}

	pub := url.Values{}
	pub.Set("creation_id", container.ID)
	var posted struct {
		ID string `json:"id"`
	}
	if err := a.postForm(ctx, "/"+igID+"/media_publish", token.AccessToken, pub, &posted); err != nil {
		return nil, err
	}
	return &publish.Result{PlatformPostID: posted.ID}, nil
}

// waitForContainer polls until the container is FINISHED. Still-processing
// after the budget returns retryable so the job re-runs later.
func (a *Adapter) waitForContainer(ctx context.Context, token, containerID string) error {
	for i := 0; i < containerPolls; i++ {
		var st struct {
			StatusCode string `json:"status_code"`
		}
		if err := a.getJSON(ctx, "/"+containerID+"?fields=status_code", token, &st); err != nil {
			return err
		}
		switch st.StatusCode {
		case "FINISHED":
			return nil
		case "ERROR", "EXPIRED":
			return publish.Terminal("container_failed", "Instagram rejected the media container", nil)
		}
		select {
		case <-ctx.Done():
			return publish.Retryable("canceled", "publish interrupted", ctx.Err())
		case <-time.After(containerPollWait):
		}
	}
	return publish.Retryable("container_processing", "Instagram is still processing the media", nil)
}

// FetchMetrics implements publish.Adapter via the media insights edge.
func (a *Adapter) FetchMetrics(ctx context.Context, token channel.Token, platformPostID string) ([]publish.Metric, error) {
	var out struct {
		Data []struct {
			Name   string `json:"name"`
			Values []struct {
				Value int64 `json:"value"`
			} `json:"values"`
		} `json:"data"`
	}
	path := "/" + platformPostID + "/insights?metric=likes,comments,shares,saved,reach"
	if err := a.getJSON(ctx, path, token.AccessToken, &out); err != nil {
		return nil, err
	}
	metrics := make([]publish.Metric, 0, len(out.Data))
	for _, d := range out.Data {
		if len(d.Values) == 0 {
			continue
		}
		metrics = append(metrics, publish.Metric{Name: d.Name, Value: d.Values[0].Value})
	}
	return metrics, nil
}

// getJSON performs a versioned GET, decoding into out and mapping errors.
func (a *Adapter) getJSON(ctx context.Context, path, token string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, a.cfg.APIBaseURL+"/"+apiVersion+path, nil)
	if err != nil {
		return fmt.Errorf("building instagram request: %w", err)
	}
	return a.do(req, token, out)
}

// postForm performs a versioned form POST, decoding into out.
func (a *Adapter) postForm(ctx context.Context, path, token string, form url.Values, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		a.cfg.APIBaseURL+"/"+apiVersion+path, strings.NewReader(form.Encode()))
	if err != nil {
		return fmt.Errorf("building instagram request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return a.do(req, token, out)
}

func (a *Adapter) do(req *http.Request, token string, out any) error {
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := a.http.Do(req)
	if err != nil {
		return publish.Retryable("network_error", "could not reach Instagram", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		if out == nil {
			return nil
		}
		if err := json.Unmarshal(body, out); err != nil {
			return publish.Terminal("bad_response", "could not parse Instagram response", err)
		}
		return nil
	}
	return classify(resp.StatusCode, body)
}

// classify maps Graph API errors onto adapter error classes.
func classify(status int, body []byte) error {
	var fb struct {
		Error struct {
			Message string `json:"message"`
			Code    int    `json:"code"`
		} `json:"error"`
	}
	_ = json.Unmarshal(body, &fb)
	msg := fb.Error.Message
	if msg == "" {
		msg = fmt.Sprintf("Instagram returned status %d", status)
	}
	switch {
	case status == http.StatusUnauthorized || fb.Error.Code == 190:
		return publish.AuthExpired(fmt.Errorf("instagram: %s", msg))
	case status == http.StatusTooManyRequests || fb.Error.Code == 4 || fb.Error.Code == 17 || fb.Error.Code == 32:
		return publish.Retryable("rate_limited", msg, nil)
	case status >= 500:
		return publish.Retryable("server_error", msg, nil)
	default:
		return publish.Terminal("instagram_error", msg, nil)
	}
}
