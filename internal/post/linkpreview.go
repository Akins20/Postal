package post

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"syscall"
	"time"
)

// Link-preview fetch limits: small and fast, it's a compose-time hint.
const (
	linkPreviewTimeout  = 4 * time.Second
	linkPreviewMaxBody  = 512 << 10 // 512 KiB of HTML is plenty for <head>
	linkPreviewMaxHops  = 3
	linkPreviewMaxField = 300 // characters per extracted field
)

// LinkPreview is the OpenGraph summary of a URL, used by the composer to
// render the platform-style link card. Empty fields were absent on the page.
type LinkPreview struct {
	URL         string `json:"url"`
	SiteName    string `json:"site_name"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Image       string `json:"image"`
}

// LinkPreviewer fetches OpenGraph metadata for compose-time link cards. The
// fetch is SSRF-guarded: http(s) only, public addresses only (checked at
// connect time, after DNS), bounded redirects, time, and body size.
type LinkPreviewer struct {
	client *http.Client
	// allowPrivate disables the address guard FOR TESTS ONLY (httptest binds
	// to loopback). Never enable it in application wiring.
	allowPrivate bool
}

// NewLinkPreviewer builds the production previewer (guards on).
func NewLinkPreviewer() *LinkPreviewer {
	p := &LinkPreviewer{}
	p.client = p.buildClient()
	return p
}

// newTestLinkPreviewer builds a previewer that may dial loopback (unit tests).
func newTestLinkPreviewer() *LinkPreviewer {
	p := &LinkPreviewer{allowPrivate: true}
	p.client = p.buildClient()
	return p
}

func (p *LinkPreviewer) buildClient() *http.Client {
	dialer := &net.Dialer{
		Timeout: linkPreviewTimeout,
		// Control runs after DNS resolution with the literal address, so a
		// hostname resolving to a private IP is still blocked.
		Control: func(_, address string, _ syscall.RawConn) error {
			if p.allowPrivate {
				return nil
			}
			host, _, err := net.SplitHostPort(address)
			if err != nil {
				return fmt.Errorf("bad dial address: %w", err)
			}
			ip := net.ParseIP(host)
			if ip == nil || !ip.IsGlobalUnicast() || ip.IsPrivate() {
				return errors.New("destination address is not publicly routable")
			}
			return nil
		},
	}
	return &http.Client{
		Timeout: linkPreviewTimeout,
		Transport: &http.Transport{
			DialContext:       dialer.DialContext,
			DisableKeepAlives: true,
		},
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= linkPreviewMaxHops {
				return errors.New("too many redirects")
			}
			if req.URL.Scheme != "http" && req.URL.Scheme != "https" {
				return errors.New("redirect to non-http scheme")
			}
			return nil
		},
	}
}

// Fetch retrieves and parses OpenGraph metadata for raw (a user-typed URL).
func (p *LinkPreviewer) Fetch(ctx context.Context, raw string) (*LinkPreview, error) {
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return nil, fmt.Errorf("not a fetchable http(s) URL")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("building preview request: %w", err)
	}
	// Some sites only serve OG tags to identified crawlers.
	req.Header.Set("User-Agent", "PostalLinkPreview/1.0 (+https://postal.example)")
	req.Header.Set("Accept", "text/html")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching link: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("link returned status %d", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "" && !strings.Contains(ct, "html") {
		return &LinkPreview{URL: u.String(), Title: u.Host}, nil // non-HTML: bare card
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, linkPreviewMaxBody))
	if err != nil {
		return nil, fmt.Errorf("reading link body: %w", err)
	}
	pv := parseOpenGraph(string(body))
	pv.URL = u.String()
	if pv.Title == "" {
		pv.Title = u.Host
	}
	return pv, nil
}

var (
	metaTagRe   = regexp.MustCompile(`(?is)<meta\s[^>]*>`)
	metaAttrRe  = regexp.MustCompile(`(?is)(property|name|content)\s*=\s*("([^"]*)"|'([^']*)')`)
	htmlTitleRe = regexp.MustCompile(`(?is)<title[^>]*>(.*?)</title>`)
)

// parseOpenGraph extracts og:* fields (plus <title> as a fallback) without an
// HTML-parser dependency; tolerant of attribute order and quoting.
func parseOpenGraph(html string) *LinkPreview {
	pv := &LinkPreview{}
	for _, tag := range metaTagRe.FindAllString(html, 200) {
		var key, content string
		for _, m := range metaAttrRe.FindAllStringSubmatch(tag, -1) {
			val := m[3] + m[4]
			switch strings.ToLower(m[1]) {
			case "property", "name":
				key = strings.ToLower(val)
			case "content":
				content = val
			}
		}
		if content == "" {
			continue
		}
		content = clampField(content)
		switch key {
		case "og:title":
			pv.Title = content
		case "og:description", "description":
			if pv.Description == "" || key == "og:description" {
				pv.Description = content
			}
		case "og:image":
			pv.Image = content
		case "og:site_name":
			pv.SiteName = content
		}
	}
	if pv.Title == "" {
		if m := htmlTitleRe.FindStringSubmatch(html); m != nil {
			pv.Title = clampField(strings.TrimSpace(m[1]))
		}
	}
	return pv
}

// clampField bounds extracted text so hostile pages can't bloat responses.
func clampField(s string) string {
	if len(s) > linkPreviewMaxField {
		return s[:linkPreviewMaxField]
	}
	return s
}
