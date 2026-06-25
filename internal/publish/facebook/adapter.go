// Package facebook implements publish.Adapter for Facebook Pages via the Meta
// Graph API. Posts go to a Page's feed (text/link), photos, or videos using the
// Page access token resolved from the connected user. Unlike Instagram, Facebook
// allows text-only posts. See docs/PLATFORMS_IG_TIKTOK.md.
package facebook

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
	platformFacebook   = "facebook"
	defaultAPIBaseURL  = "https://graph.facebook.com"
	defaultAuthBaseURL = "https://www.facebook.com"
	apiVersion         = "v21.0"

	maxMessageRunes = 63206 // Facebook's documented status length cap
	maxImageBytes   = 10 << 20
	maxVideoBytes   = 4 << 30
	oauthScopes     = "pages_show_list,pages_manage_posts,pages_read_engagement,public_profile"
)

// Config configures the Facebook adapter. Base URLs are overridable so dev and
// tests point at the simulator.
type Config struct {
	ClientID     string
	ClientSecret string
	RedirectURI  string
	APIBaseURL   string
	AuthBaseURL  string
	HTTPClient   *http.Client
}

// Adapter implements publish.Adapter for Facebook Pages (Meta Graph API).
type Adapter struct {
	cfg  Config
	http *http.Client
}

// New builds a Facebook adapter, applying default hosts and client.
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
func (a *Adapter) Platform() string { return platformFacebook }

// Constraints implements publish.Adapter. Facebook allows text-only posts.
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

// Validate implements publish.Adapter: the message is capped and a post may
// carry at most one image or video (multi-photo posts are a later enhancement).
func (a *Adapter) Validate(v publish.PostVariant) error {
	if utf8.RuneCountInString(v.Text) > maxMessageRunes {
		return publish.Terminal("message_too_long",
			fmt.Sprintf("Facebook posts are limited to %d characters", maxMessageRunes), nil)
	}
	if strings.TrimSpace(v.Text) == "" && len(v.Media) == 0 {
		return publish.Terminal("empty_post", "a Facebook post needs text or media", nil)
	}
	if len(v.Media) > 1 {
		return publish.Terminal("too_many_media",
			"Facebook posts support one image or video for now", nil)
	}
	if len(v.Media) == 1 {
		m := v.Media[0]
		if m.Kind == publish.MediaImage && m.Bytes > maxImageBytes {
			return publish.Terminal("image_too_large", "Facebook images are limited to 10 MiB", nil)
		}
		if m.Kind == publish.MediaVideo && m.Bytes > maxVideoBytes {
			return publish.Terminal("video_too_large", "Facebook videos are limited to 4 GiB", nil)
		}
	}
	return nil
}

// AuthURL implements channel.OAuthProvider. Meta's dialog has no PKCE; the
// challenge parameter is accepted and ignored.
func (a *Adapter) AuthURL(state, _, redirectURI string) string {
	if redirectURI == "" {
		redirectURI = a.cfg.RedirectURI
	}
	v := url.Values{}
	v.Set("client_id", a.cfg.ClientID)
	v.Set("redirect_uri", redirectURI)
	v.Set("state", state)
	v.Set("scope", oauthScopes)
	v.Set("response_type", "code")
	return a.cfg.AuthBaseURL + "/" + apiVersion + "/dialog/oauth?" + v.Encode()
}

// ExchangeCode implements channel.OAuthProvider: code -> short-lived token ->
// long-lived (~60 day) token. Meta has no refresh grant; the long-lived token
// is stored as the "refresh token" and re-exchanged before expiry.
func (a *Adapter) ExchangeCode(ctx context.Context, code, _, redirectURI string) (*channel.Token, error) {
	if redirectURI == "" {
		redirectURI = a.cfg.RedirectURI
	}
	v := url.Values{}
	v.Set("client_id", a.cfg.ClientID)
	v.Set("client_secret", a.cfg.ClientSecret)
	v.Set("redirect_uri", redirectURI)
	v.Set("code", code)
	var short struct {
		AccessToken string `json:"access_token"`
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
		RefreshToken: out.AccessToken,
		Scopes:       strings.Split(oauthScopes, ","),
		ExpiresAt:    expires,
	}, nil
}

// page holds the connected Facebook Page's id and its page access token (used
// for publishing, distinct from the user token).
type page struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Token string `json:"access_token"`
}

// firstPage resolves the user's first managed Page (id, name, page token).
func (a *Adapter) firstPage(ctx context.Context, userToken string) (page, error) {
	var pages struct {
		Data []page `json:"data"`
	}
	if err := a.getJSON(ctx, "/me/accounts?fields=id,name,access_token", userToken, &pages); err != nil {
		return page{}, err
	}
	if len(pages.Data) == 0 {
		return page{}, publish.Terminal("no_facebook_page",
			"no Facebook Page is available; create or grant access to a Page", nil)
	}
	return pages.Data[0], nil
}

// Account implements channel.OAuthProvider: the connected account is the Page.
func (a *Adapter) Account(ctx context.Context, accessToken string) (*channel.Account, error) {
	p, err := a.firstPage(ctx, accessToken)
	if err != nil {
		return nil, err
	}
	return &channel.Account{ID: p.ID, Handle: p.Name, DisplayName: p.Name}, nil
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
		return fmt.Errorf("revoking facebook token: %w", err)
	}
	_ = resp.Body.Close()
	return nil
}

// Publish implements publish.Adapter. Text/link posts go to the Page feed;
// a single image to /photos and a single video to /videos, using the Page token.
func (a *Adapter) Publish(ctx context.Context, token channel.Token, v publish.PostVariant) (*publish.Result, error) {
	if err := a.Validate(v); err != nil {
		return nil, err
	}
	p, err := a.firstPage(ctx, token.AccessToken)
	if err != nil {
		return nil, err
	}

	form := url.Values{}
	var path string
	switch {
	case len(v.Media) == 0:
		path = "/" + p.ID + "/feed"
		form.Set("message", v.Text)
	case v.Media[0].Kind == publish.MediaVideo:
		if v.Media[0].URL == "" {
			return nil, publish.Retryable("media_url_unavailable",
				"no public media URL available for Facebook (storage presigning failed)", nil)
		}
		path = "/" + p.ID + "/videos"
		form.Set("file_url", v.Media[0].URL)
		form.Set("description", v.Text)
	default:
		if v.Media[0].URL == "" {
			return nil, publish.Retryable("media_url_unavailable",
				"no public media URL available for Facebook (storage presigning failed)", nil)
		}
		path = "/" + p.ID + "/photos"
		form.Set("url", v.Media[0].URL)
		form.Set("caption", v.Text)
	}

	var posted struct {
		ID     string `json:"id"`
		PostID string `json:"post_id"`
	}
	if err := a.postForm(ctx, path, p.Token, form, &posted); err != nil {
		return nil, err
	}
	id := posted.PostID
	if id == "" {
		id = posted.ID
	}
	return &publish.Result{PlatformPostID: id}, nil
}

// FetchMetrics implements publish.Adapter via Page post insights.
func (a *Adapter) FetchMetrics(ctx context.Context, token channel.Token, platformPostID string) ([]publish.Metric, error) {
	p, err := a.firstPage(ctx, token.AccessToken)
	if err != nil {
		return nil, err
	}
	var out struct {
		Data []struct {
			Name   string `json:"name"`
			Values []struct {
				Value int64 `json:"value"`
			} `json:"values"`
		} `json:"data"`
	}
	path := "/" + platformPostID + "/insights?metric=post_impressions,post_engaged_users,post_reactions_by_type_total"
	if err := a.getJSON(ctx, path, p.Token, &out); err != nil {
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

func (a *Adapter) getJSON(ctx context.Context, path, token string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, a.cfg.APIBaseURL+"/"+apiVersion+path, nil)
	if err != nil {
		return fmt.Errorf("building facebook request: %w", err)
	}
	return a.do(req, token, out)
}

func (a *Adapter) postForm(ctx context.Context, path, token string, form url.Values, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		a.cfg.APIBaseURL+"/"+apiVersion+path, strings.NewReader(form.Encode()))
	if err != nil {
		return fmt.Errorf("building facebook request: %w", err)
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
		return publish.Retryable("network_error", "could not reach Facebook", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		if out == nil {
			return nil
		}
		if err := json.Unmarshal(body, out); err != nil {
			return publish.Terminal("bad_response", "could not parse Facebook response", err)
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
		msg = fmt.Sprintf("Facebook returned status %d", status)
	}
	switch {
	case status == http.StatusUnauthorized || fb.Error.Code == 190:
		return publish.AuthExpired(fmt.Errorf("facebook: %s", msg))
	case status == http.StatusTooManyRequests || fb.Error.Code == 4 || fb.Error.Code == 17 || fb.Error.Code == 32:
		return publish.Retryable("rate_limited", msg, nil)
	case status >= 500:
		return publish.Retryable("server_error", msg, nil)
	default:
		return publish.Terminal("facebook_error", msg, nil)
	}
}
