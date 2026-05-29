package post

import (
	"net/url"
	"regexp"
	"strings"
)

// linkPattern matches http(s) URLs in post text for UTM tagging. Restricted to
// printable ASCII so it doesn't run into adjacent non-ASCII characters.
var linkPattern = regexp.MustCompile(`https?://[!-~]+`)

// trailingPunct are characters commonly adjacent to a URL in prose (sentence
// punctuation, closing brackets/quotes) that should NOT be absorbed into the URL
// when tagging.
const trailingPunct = `.,;:!?)]}'"`

// ApplyUTM appends the given UTM parameters (e.g. utm_source, utm_medium,
// utm_campaign) to every http(s) URL in text, preserving any existing query.
// Empty values are skipped; existing keys are overwritten. Trailing prose
// punctuation is preserved outside the URL. Non-URL text is returned unchanged.
// (Link shortening is a separate, later concern.)
func ApplyUTM(text string, params map[string]string) string {
	if len(params) == 0 {
		return text
	}
	return linkPattern.ReplaceAllStringFunc(text, func(match string) string {
		// Peel trailing punctuation back off the match so "see https://x.com."
		// tags the URL and keeps the period as sentence text.
		trimmed := strings.TrimRight(match, trailingPunct)
		suffix := match[len(trimmed):]

		u, err := url.Parse(trimmed)
		if err != nil {
			return match
		}
		q := u.Query()
		for k, v := range params {
			if v != "" {
				q.Set(k, v)
			}
		}
		u.RawQuery = q.Encode()
		return u.String() + suffix
	})
}
