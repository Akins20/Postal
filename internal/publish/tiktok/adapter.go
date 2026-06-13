// Package tiktok implements publish.Adapter for TikTok's Content Posting API.
// Videos upload as FILE_UPLOAD (single chunk PUT to the returned upload_url,
// avoiding TikTok's domain verification for pulled URLs); photo posts pull
// from presigned URLs. A creator_info query precedes every post, per the API
// contract. Unaudited API clients can only post privately - the UI says so.
// See docs/PLATFORMS_IG_TIKTOK.md.
package tiktok

import (
	"bytes"
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
	platformTikTok     = "tiktok"
	defaultAPIBaseURL  = "https://open.tiktokapis.com"
	defaultAuthBaseURL = "https://www.tiktok.com"

	maxCaptionRunes = 2200
	maxPhotos       = 10
	maxImageBytes   = 20 << 20
	maxVideoBytes   = 512 << 20
	oauthScopes     = "user.info.basic,video.publish,video.list"

	// statusPolls bounds publish-status checks per attempt; still-processing
	// posts return retryable so asynq re-runs the job later.
	statusPolls = 10
)

// statusPollWait spaces publish-status checks. A var so tests can lower it.
var statusPollWait = 3 * time.Second

// Config configures the TikTok adapter. Base URLs are overridable so dev and
// tests point at the simulator.
type Config struct {
	ClientKey    string
	ClientSecret string
	RedirectURI  string
	APIBaseURL   string
	AuthBaseURL  string
	HTTPClient   *http.Client
}

// Adapter implements publish.Adapter for TikTok.
type Adapter struct {
	cfg  Config
	http *http.Client
}

// New builds a TikTok adapter, applying default hosts and HTTP client.
func New(cfg Config) *Adapter {
	if cfg.APIBaseURL == "" {
		cfg.APIBaseURL = defaultAPIBaseURL
	}
	if cfg.AuthBaseURL == "" {
		cfg.AuthBaseURL = defaultAuthBaseURL
	}
	if cfg.HTTPClient == nil {
		cfg.HTTPClient = &http.Client{Timeout: 60 * time.Second}
	}
	return &Adapter{cfg: cfg, http: cfg.HTTPClient}
}

// Platform implements channel.OAuthProvider.
func (a *Adapter) Platform() string { return platformTikTok }

// Constraints implements publish.Adapter.
func (a *Adapter) Constraints() publish.Constraints {
	return publish.Constraints{
		RequiresMedia:      true,
		MaxWeightedTextLen: maxCaptionRunes,
		MaxImages:          maxPhotos,
		MaxVideos:          1,
		MaxImageBytes:      maxImageBytes,
		MaxVideoBytes:      maxVideoBytes,
	}
}

// Validate implements publish.Adapter: a post is one video OR up to ten
// photos, never text-only, never GIFs.
func (a *Adapter) Validate(v publish.PostVariant) error {
	if utf8.RuneCountInString(v.Text) > maxCaptionRunes {
		return publish.Terminal("caption_too_long",
			fmt.Sprintf("TikTok captions are limited to %d characters", maxCaptionRunes), nil)
	}
	if len(v.Media) == 0 {
		return publish.Terminal("media_required",
			"TikTok does not allow text-only posts; attach a video or photos", nil)
	}
	videos, photos := 0, 0
	for _, m := range v.Media {
		switch m.Kind {
		case publish.MediaVideo:
			videos++
			if m.Bytes > maxVideoBytes {
				return publish.Terminal("video_too_large", "TikTok videos are limited to 512 MiB here", nil)
			}
		case publish.MediaImage:
			photos++
			if m.Bytes > maxImageBytes {
				return publish.Terminal("image_too_large", "TikTok photos are limited to 20 MiB", nil)
			}
		default:
			return publish.Terminal("unsupported_media", "TikTok accepts videos and photos only", nil)
		}
	}
	if videos > 1 || (videos == 1 && photos > 0) {
		return publish.Terminal("mixed_media", "TikTok posts are one video OR photos, not both", nil)
	}
	if photos > maxPhotos {
		return publish.Terminal("too_many_photos",
			fmt.Sprintf("TikTok photo posts are limited to %d images", maxPhotos), nil)
	}
	return nil
}

// AuthURL implements channel.OAuthProvider (PKCE S256, client_key param).
func (a *Adapter) AuthURL(state, codeChallenge, redirectURI string) string {
	if redirectURI == "" {
		redirectURI = a.cfg.RedirectURI
	}
	v := url.Values{}
	v.Set("client_key", a.cfg.ClientKey)
	v.Set("response_type", "code")
	v.Set("scope", oauthScopes)
	v.Set("redirect_uri", redirectURI)
	v.Set("state", state)
	v.Set("code_challenge", codeChallenge)
	v.Set("code_challenge_method", "S256")
	return a.cfg.AuthBaseURL + "/v2/auth/authorize/?" + v.Encode()
}

// ExchangeCode implements channel.OAuthProvider.
func (a *Adapter) ExchangeCode(ctx context.Context, code, codeVerifier, redirectURI string) (*channel.Token, error) {
	if redirectURI == "" {
		redirectURI = a.cfg.RedirectURI
	}
	form := url.Values{}
	form.Set("client_key", a.cfg.ClientKey)
	form.Set("client_secret", a.cfg.ClientSecret)
	form.Set("code", code)
	form.Set("grant_type", "authorization_code")
	form.Set("redirect_uri", redirectURI)
	form.Set("code_verifier", codeVerifier)
	return a.tokenRequest(ctx, form)
}

// RefreshToken implements channel.OAuthProvider.
func (a *Adapter) RefreshToken(ctx context.Context, refreshToken string) (*channel.Token, error) {
	form := url.Values{}
	form.Set("client_key", a.cfg.ClientKey)
	form.Set("client_secret", a.cfg.ClientSecret)
	form.Set("grant_type", "refresh_token")
	form.Set("refresh_token", refreshToken)
	return a.tokenRequest(ctx, form)
}

func (a *Adapter) tokenRequest(ctx context.Context, form url.Values) (*channel.Token, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		a.cfg.APIBaseURL+"/v2/oauth/token/", strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("building tiktok token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	var out struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int64  `json:"expires_in"`
		Scope        string `json:"scope"`
		Error        string `json:"error"`
		ErrorDesc    string `json:"error_description"`
	}
	if err := a.doJSON(req, "", &out); err != nil {
		return nil, err
	}
	if out.Error != "" || out.AccessToken == "" {
		return nil, publish.Terminal("oauth_failed", "TikTok rejected the authorization: "+out.ErrorDesc, nil)
	}
	return &channel.Token{
		AccessToken:  out.AccessToken,
		RefreshToken: out.RefreshToken,
		Scopes:       strings.Split(out.Scope, ","),
		ExpiresAt:    time.Now().Add(time.Duration(out.ExpiresIn) * time.Second),
	}, nil
}

// Account implements channel.OAuthProvider via /v2/user/info/.
func (a *Adapter) Account(ctx context.Context, accessToken string) (*channel.Account, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		a.cfg.APIBaseURL+"/v2/user/info/?fields=open_id,display_name,username", nil)
	if err != nil {
		return nil, fmt.Errorf("building tiktok request: %w", err)
	}
	var out struct {
		Data struct {
			User struct {
				OpenID      string `json:"open_id"`
				DisplayName string `json:"display_name"`
				Username    string `json:"username"`
			} `json:"user"`
		} `json:"data"`
	}
	if err := a.doJSON(req, accessToken, &out); err != nil {
		return nil, err
	}
	handle := out.Data.User.Username
	if handle != "" {
		handle = "@" + handle
	}
	return &channel.Account{ID: out.Data.User.OpenID, Handle: handle, DisplayName: out.Data.User.DisplayName}, nil
}

// Revoke implements channel.OAuthProvider (best effort).
func (a *Adapter) Revoke(ctx context.Context, token string) error {
	form := url.Values{}
	form.Set("client_key", a.cfg.ClientKey)
	form.Set("client_secret", a.cfg.ClientSecret)
	form.Set("token", token)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		a.cfg.APIBaseURL+"/v2/oauth/revoke/", strings.NewReader(form.Encode()))
	if err != nil {
		return fmt.Errorf("building revoke request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := a.http.Do(req)
	if err != nil {
		return fmt.Errorf("revoking tiktok token: %w", err)
	}
	_ = resp.Body.Close()
	return nil
}

// Publish implements publish.Adapter: creator_info first, then a direct post
// (video FILE_UPLOAD or photo PULL_FROM_URL), then status polling.
func (a *Adapter) Publish(ctx context.Context, token channel.Token, v publish.PostVariant) (*publish.Result, error) {
	if err := a.Validate(v); err != nil {
		return nil, err
	}
	privacy, err := a.creatorPrivacy(ctx, token.AccessToken)
	if err != nil {
		return nil, err
	}

	var publishID string
	if v.Media[0].Kind == publish.MediaVideo {
		publishID, err = a.postVideo(ctx, token.AccessToken, v, privacy)
	} else {
		publishID, err = a.postPhotos(ctx, token.AccessToken, v, privacy)
	}
	if err != nil {
		return nil, err
	}
	postID, err := a.waitForPublish(ctx, token.AccessToken, publishID)
	if err != nil {
		return nil, err
	}
	return &publish.Result{PlatformPostID: postID}, nil
}

// creatorPrivacy queries creator_info (required pre-post) and picks the most
// public level the account offers; unaudited apps only offer SELF_ONLY.
func (a *Adapter) creatorPrivacy(ctx context.Context, token string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		a.cfg.APIBaseURL+"/v2/post/publish/creator_info/query/", strings.NewReader("{}"))
	if err != nil {
		return "", fmt.Errorf("building creator_info request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	var out struct {
		Data struct {
			PrivacyLevelOptions []string `json:"privacy_level_options"`
		} `json:"data"`
	}
	if err := a.doJSON(req, token, &out); err != nil {
		return "", err
	}
	for _, level := range out.Data.PrivacyLevelOptions {
		if level == "PUBLIC_TO_EVERYONE" {
			return level, nil
		}
	}
	if len(out.Data.PrivacyLevelOptions) > 0 {
		return out.Data.PrivacyLevelOptions[0], nil
	}
	return "SELF_ONLY", nil
}

func (a *Adapter) postVideo(ctx context.Context, token string, v publish.PostVariant, privacy string) (string, error) {
	m := v.Media[0]
	size := int64(len(m.Data))
	if size == 0 {
		return "", publish.Retryable("media_bytes_unavailable", "video bytes were not loaded", nil)
	}
	body := map[string]any{
		"post_info": map[string]any{"title": v.Text, "privacy_level": privacy},
		"source_info": map[string]any{
			"source": "FILE_UPLOAD", "video_size": size,
			"chunk_size": size, "total_chunk_count": 1,
		},
	}
	var out struct {
		Data struct {
			PublishID string `json:"publish_id"`
			UploadURL string `json:"upload_url"`
		} `json:"data"`
	}
	if err := a.postJSON(ctx, "/v2/post/publish/video/init/", token, body, &out); err != nil {
		return "", err
	}
	if out.Data.UploadURL == "" {
		return "", publish.Terminal("no_upload_url", "TikTok did not return an upload URL", nil)
	}

	up, err := http.NewRequestWithContext(ctx, http.MethodPut, out.Data.UploadURL, bytes.NewReader(m.Data))
	if err != nil {
		return "", fmt.Errorf("building upload request: %w", err)
	}
	up.Header.Set("Content-Type", m.MIME)
	up.Header.Set("Content-Range", fmt.Sprintf("bytes 0-%d/%d", size-1, size))
	resp, err := a.http.Do(up)
	if err != nil {
		return "", publish.Retryable("upload_failed", "could not upload the video to TikTok", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode >= 300 {
		return "", publish.Retryable("upload_rejected",
			fmt.Sprintf("TikTok upload returned %d", resp.StatusCode), nil)
	}
	return out.Data.PublishID, nil
}

func (a *Adapter) postPhotos(ctx context.Context, token string, v publish.PostVariant, privacy string) (string, error) {
	urls := make([]string, 0, len(v.Media))
	for _, m := range v.Media {
		if m.URL == "" {
			return "", publish.Retryable("media_url_unavailable",
				"no public photo URL available for TikTok (storage presigning failed)", nil)
		}
		urls = append(urls, m.URL)
	}
	body := map[string]any{
		"post_info":  map[string]any{"title": v.Text, "privacy_level": privacy},
		"media_type": "PHOTO",
		"post_mode":  "DIRECT_POST",
		"source_info": map[string]any{
			"source": "PULL_FROM_URL", "photo_images": urls, "photo_cover_index": 0,
		},
	}
	var out struct {
		Data struct {
			PublishID string `json:"publish_id"`
		} `json:"data"`
	}
	if err := a.postJSON(ctx, "/v2/post/publish/content/init/", token, body, &out); err != nil {
		return "", err
	}
	return out.Data.PublishID, nil
}

// waitForPublish polls status/fetch until the post completes. Still
// processing after the budget returns retryable for a later re-run.
func (a *Adapter) waitForPublish(ctx context.Context, token, publishID string) (string, error) {
	for i := 0; i < statusPolls; i++ {
		var out struct {
			Data struct {
				Status string `json:"status"`
				// TikTok's API genuinely misspells this field name.
				PublicalyAvailablePost []int64 `json:"publicaly_available_post_id"` //nolint:misspell // upstream field name
				FailReason             string  `json:"fail_reason"`
			} `json:"data"`
		}
		if err := a.postJSON(ctx, "/v2/post/publish/status/fetch/", token,
			map[string]string{"publish_id": publishID}, &out); err != nil {
			return "", err
		}
		switch out.Data.Status {
		case "PUBLISH_COMPLETE":
			if len(out.Data.PublicalyAvailablePost) > 0 {
				return fmt.Sprintf("%d", out.Data.PublicalyAvailablePost[0]), nil
			}
			return publishID, nil // private posts expose no public id
		case "FAILED":
			return "", publish.Terminal("publish_failed", "TikTok rejected the post: "+out.Data.FailReason, nil)
		}
		select {
		case <-ctx.Done():
			return "", publish.Retryable("canceled", "publish interrupted", ctx.Err())
		case <-time.After(statusPollWait):
		}
	}
	return "", publish.Retryable("still_processing", "TikTok is still processing the post", nil)
}

// FetchMetrics implements publish.Adapter via /v2/video/query/.
func (a *Adapter) FetchMetrics(ctx context.Context, token channel.Token, platformPostID string) ([]publish.Metric, error) {
	body := map[string]any{"filters": map[string]any{"video_ids": []string{platformPostID}}}
	var out struct {
		Data struct {
			Videos []struct {
				LikeCount    int64 `json:"like_count"`
				CommentCount int64 `json:"comment_count"`
				ShareCount   int64 `json:"share_count"`
				ViewCount    int64 `json:"view_count"`
			} `json:"videos"`
		} `json:"data"`
	}
	path := "/v2/video/query/?fields=id,like_count,comment_count,share_count,view_count"
	if err := a.postJSON(ctx, path, token.AccessToken, body, &out); err != nil {
		return nil, err
	}
	if len(out.Data.Videos) == 0 {
		return nil, publish.Terminal("post_not_found", "TikTok returned no data for this post", nil)
	}
	vd := out.Data.Videos[0]
	return []publish.Metric{
		{Name: "views", Value: vd.ViewCount},
		{Name: "likes", Value: vd.LikeCount},
		{Name: "comments", Value: vd.CommentCount},
		{Name: "shares", Value: vd.ShareCount},
	}, nil
}

func (a *Adapter) postJSON(ctx context.Context, path, token string, body any, out any) error {
	raw, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("encoding tiktok request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.cfg.APIBaseURL+path, bytes.NewReader(raw))
	if err != nil {
		return fmt.Errorf("building tiktok request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	return a.doJSON(req, token, out)
}

func (a *Adapter) doJSON(req *http.Request, token string, out any) error {
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := a.http.Do(req)
	if err != nil {
		return publish.Retryable("network_error", "could not reach TikTok", err)
	}
	defer func() { _ = resp.Body.Close() }()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		if out == nil {
			return nil
		}
		if err := json.Unmarshal(raw, out); err != nil {
			return publish.Terminal("bad_response", "could not parse TikTok response", err)
		}
		return nil
	}
	return classify(resp.StatusCode, raw)
}

// classify maps TikTok error envelopes onto adapter error classes.
func classify(status int, body []byte) error {
	var tk struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	_ = json.Unmarshal(body, &tk)
	msg := tk.Error.Message
	if msg == "" {
		msg = fmt.Sprintf("TikTok returned status %d", status)
	}
	switch {
	case status == http.StatusUnauthorized || tk.Error.Code == "access_token_invalid":
		return publish.AuthExpired(fmt.Errorf("tiktok: %s", msg))
	case status == http.StatusTooManyRequests || tk.Error.Code == "rate_limit_exceeded":
		return publish.Retryable("rate_limited", msg, nil)
	case status >= 500:
		return publish.Retryable("server_error", msg, nil)
	default:
		return publish.Terminal("tiktok_error", msg, nil)
	}
}
