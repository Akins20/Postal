// Package integration manages third-party workspace integrations. The first
// provider is OGShortener (ogshortener.site): URL shortening with click
// analytics. API access requires the workspace's own OGShortener key (their
// Pro plan); the key is envelope-encrypted at rest like channel tokens.
package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"time"
)

// ProviderOGShortener is the provider key for ogshortener.site.
const ProviderOGShortener = "ogshortener"

// defaultOGShortenerAPIBase is the live API host (overridable for tests).
const defaultOGShortenerAPIBase = "https://ogshortener.site"

// urlRe finds shortenable web links in post text.
var urlRe = regexp.MustCompile(`https?://[^\s)]+`)

// OGShortenerClient calls the OGShortener REST API (X-API-Key auth).
type OGShortenerClient struct {
	apiBase string
	http    *http.Client
}

// NewOGShortenerClient builds a client. apiBase "" means the real service.
func NewOGShortenerClient(apiBase string, client *http.Client) *OGShortenerClient {
	if apiBase == "" {
		apiBase = defaultOGShortenerAPIBase
	}
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	return &OGShortenerClient{apiBase: apiBase, http: client}
}

// Verify checks an API key by listing links (no quota consumed).
func (c *OGShortenerClient) Verify(ctx context.Context, apiKey string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.apiBase+"/api/v1/links?limit=1", nil)
	if err != nil {
		return fmt.Errorf("building verify request: %w", err)
	}
	req.Header.Set("X-API-Key", apiKey)
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("calling ogshortener: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return fmt.Errorf("ogshortener rejected the API key")
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("ogshortener verify returned %d", resp.StatusCode)
	}
	return nil
}

// Shorten creates one short link and returns its short URL.
func (c *OGShortenerClient) Shorten(ctx context.Context, apiKey, longURL string) (string, error) {
	body, err := json.Marshal(map[string]string{"url": longURL})
	if err != nil {
		return "", fmt.Errorf("encoding shorten request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.apiBase+"/api/v1/links", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("building shorten request: %w", err)
	}
	req.Header.Set("X-API-Key", apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("calling ogshortener: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("ogshortener returned %d", resp.StatusCode)
	}
	var out struct {
		ShortURL string `json:"shortUrl"`
		Data     struct {
			ShortURL string `json:"shortUrl"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return "", fmt.Errorf("parsing ogshortener response: %w", err)
	}
	short := out.ShortURL
	if short == "" {
		short = out.Data.ShortURL
	}
	if short == "" {
		return "", fmt.Errorf("ogshortener response had no shortUrl")
	}
	return short, nil
}

// ShortenText replaces every link in text with its shortened form. Duplicate
// URLs shorten once. Errors abort (better an unshortened post than a half-
// rewritten one).
func (c *OGShortenerClient) ShortenText(ctx context.Context, apiKey, text string) (string, error) {
	urls := urlRe.FindAllString(text, -1)
	if len(urls) == 0 {
		return text, nil
	}
	shortened := map[string]string{}
	for _, u := range urls {
		if _, done := shortened[u]; done {
			continue
		}
		short, err := c.Shorten(ctx, apiKey, u)
		if err != nil {
			return "", fmt.Errorf("shortening %s: %w", u, err)
		}
		shortened[u] = short
	}
	return urlRe.ReplaceAllStringFunc(text, func(u string) string {
		if s, ok := shortened[u]; ok {
			return s
		}
		return u
	}), nil
}
