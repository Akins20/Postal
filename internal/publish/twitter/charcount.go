// Package twitter implements the X/Twitter PlatformAdapter (OAuth, validation,
// publishing with chunked media upload, and metrics) against X API v2. The
// adapter takes injectable base URLs so tests run against the local simulator;
// see docs/X_TWITTER_INTEGRATION.md for the verified spec.
package twitter

import (
	"regexp"

	"golang.org/x/text/unicode/norm"
)

// X weighted-counting constants (twitter-text v3 defaults). A post's weighted
// length must not exceed maxWeightedLen. URLs are charged a fixed weight
// regardless of their real length; CJK/emoji/most non-Latin runes weight 2.
const (
	maxWeightedLen = 280
	urlWeight      = 23
	heavyWeight    = 2
	lightWeight    = 1
)

// urlPattern matches http(s):// and www. URLs for fixed-weight counting. The URL
// body is restricted to printable ASCII ([!-~]) so a non-ASCII rune glued
// directly after a URL (e.g. an emoji with no separating space) is NOT swallowed
// into the URL span — otherwise those heavy-weight runes would be undercounted as
// part of the fixed 23-char URL, letting an over-limit post slip past pre-flight.
// Pragmatic approximation of twitter-text; the X API remains authoritative.
var urlPattern = regexp.MustCompile(`(?i)\b(?:https?://|www\.)[!-~]*`)

// weightedLength returns X's weighted character count of text: NFC-normalized,
// each matched URL counted as urlWeight, and remaining runes weighted 1 or 2.
func weightedLength(text string) int {
	text = norm.NFC.String(text)

	total := 0
	urls := urlPattern.FindAllString(text, -1)
	total += len(urls) * urlWeight
	remaining := urlPattern.ReplaceAllString(text, "")

	for _, r := range remaining {
		total += runeWeight(r)
	}
	return total
}

// runeWeight returns the weighted length of a single rune per the twitter-text
// v3 default ranges (weight 1 for ranges 0x0000–0x10FF, 0x2000–0x200D,
// 0x2010–0x201F, 0x2032–0x2037; weight 2 otherwise).
func runeWeight(r rune) int {
	switch {
	case r <= 0x10FF:
		return lightWeight
	case r >= 0x2000 && r <= 0x200D:
		return lightWeight
	case r >= 0x2010 && r <= 0x201F:
		return lightWeight
	case r >= 0x2032 && r <= 0x2037:
		return lightWeight
	default:
		return heavyWeight
	}
}
