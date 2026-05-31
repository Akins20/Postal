package twitter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/Akins20/postal/internal/publish"
)

// Platform key for the X/Twitter adapter.
const platformTwitter = "twitter"

// OAuth scopes requested for posting on behalf of a user (see
// docs/X_TWITTER_INTEGRATION.md §3).
const oauthScopes = "tweet.read tweet.write users.read media.write offline.access"

// Default X API hosts. Overridable via Config for tests (point at the simulator).
const (
	defaultAPIBaseURL  = "https://api.x.com"
	defaultAuthBaseURL = "https://x.com"
)

// Media size limits (bytes) and per-post counts, per the verified X spec.
// NOTE: the per-kind byte caps are mirrored in internal/media/types.go for
// upload-time validation; keep the two in sync.
const (
	maxImageBytes = 5 << 20   // 5 MiB
	maxGIFBytes   = 15 << 20  // 15 MiB
	maxVideoBytes = 512 << 20 // 512 MiB
	maxImages     = 4
	maxGIFs       = 1
	maxVideos     = 1
)

// Config configures the X adapter. ClientID/Secret/RedirectURI come from the X
// developer app; the base URLs default to production X but are injectable so
// tests run against the local simulator. HTTPClient defaults to a sane client.
type Config struct {
	ClientID     string
	ClientSecret string
	RedirectURI  string
	APIBaseURL   string // token, tweets, users, media, metrics
	AuthBaseURL  string // authorize page
	HTTPClient   *http.Client
}

// Adapter implements publish.Adapter for X/Twitter (API v2).
type Adapter struct {
	cfg  Config
	http *http.Client
}

// New builds an X adapter, applying default hosts and HTTP client when unset.
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

// compile-time assertion that the adapter satisfies the publish contract.
var _ publish.Adapter = (*Adapter)(nil)

// Platform returns the platform key.
func (a *Adapter) Platform() string { return platformTwitter }

// Constraints returns X's publishing limits.
func (a *Adapter) Constraints() publish.Constraints {
	return publish.Constraints{
		MaxWeightedTextLen: maxWeightedLen,
		URLWeight:          urlWeight,
		MaxImages:          maxImages,
		MaxGifs:            maxGIFs,
		MaxVideos:          maxVideos,
		MaxImageBytes:      maxImageBytes,
		MaxGIFBytes:        maxGIFBytes,
		MaxVideoBytes:      maxVideoBytes,
	}
}

// getJSON performs a Bearer-authenticated GET and decodes the JSON response.
func (a *Adapter) getJSON(ctx context.Context, url, bearer string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return publish.Terminal("request_build_failed", "could not build request", err)
	}
	req.Header.Set("Authorization", "Bearer "+bearer)
	return a.do(req, out)
}

// postJSON performs a Bearer-authenticated JSON POST and decodes the response.
func (a *Adapter) postJSON(ctx context.Context, url, bearer string, body, out any) error {
	buf, err := json.Marshal(body)
	if err != nil {
		return publish.Terminal("encode_failed", "could not encode request body", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(buf))
	if err != nil {
		return publish.Terminal("request_build_failed", "could not build request", err)
	}
	req.Header.Set("Authorization", "Bearer "+bearer)
	req.Header.Set("Content-Type", "application/json")
	return a.do(req, out)
}

// do executes a request, decoding a 2xx JSON body into out, and mapping non-2xx
// responses (and transport errors) to a classified publish.Error.
func (a *Adapter) do(req *http.Request, out any) error {
	resp, err := a.http.Do(req)
	if err != nil {
		// Transport errors (timeouts, connection resets) are retryable.
		return publish.Retryable("network_error", "request failed", err)
	}
	defer func() { _ = resp.Body.Close() }()

	data, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		if out == nil || len(data) == 0 {
			return nil
		}
		if err := json.Unmarshal(data, out); err != nil {
			return publish.Terminal("decode_failed", "could not decode response", err)
		}
		return nil
	}
	return classifyHTTPError(resp, data)
}

// classifyHTTPError maps an X error response to a publish.Error with the right
// retry class. See docs/X_TWITTER_INTEGRATION.md §5.
func classifyHTTPError(resp *http.Response, body []byte) *publish.Error {
	switch {
	case resp.StatusCode == http.StatusTooManyRequests:
		return publish.RateLimited(retryAfterFromHeaders(resp.Header), errBody(body))
	case resp.StatusCode == http.StatusUnauthorized:
		return publish.AuthExpired(errBody(body))
	case resp.StatusCode == http.StatusForbidden:
		// 403 covers duplicate content, suspension, forbidden content — terminal.
		return publish.Terminal(forbiddenCode(body), "request forbidden by platform", errBody(body))
	case resp.StatusCode >= 500:
		return publish.Retryable("server_error", "platform server error", errBody(body))
	default:
		// 400/422 and other 4xx: invalid request — terminal, fail fast.
		return publish.Terminal("invalid_request", "platform rejected the request", errBody(body))
	}
}

// retryAfterFromHeaders derives a backoff from x-rate-limit-reset (unix secs).
// Falls back to a small default when the header is absent.
func retryAfterFromHeaders(h http.Header) time.Duration {
	const fallback = 15 * time.Second
	reset := h.Get("x-rate-limit-reset")
	if reset == "" {
		return fallback
	}
	ts, err := strconv.ParseInt(reset, 10, 64)
	if err != nil {
		return fallback
	}
	d := time.Until(time.Unix(ts, 0))
	if d <= 0 {
		return time.Second
	}
	return d
}

// forbiddenCode extracts a specific code for 403s (e.g. duplicate) for clearer
// errors, defaulting to "forbidden".
func forbiddenCode(body []byte) string {
	if bytes.Contains(bytes.ToLower(body), []byte("duplicate")) {
		return "duplicate"
	}
	return "forbidden"
}

// errBody wraps a truncated response body as an error for logging context.
func errBody(body []byte) error {
	if len(body) == 0 {
		return nil
	}
	const max = 512
	if len(body) > max {
		body = body[:max]
	}
	return fmt.Errorf("platform response: %s", body)
}
