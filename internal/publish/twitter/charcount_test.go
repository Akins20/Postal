package twitter

import (
	"strings"
	"testing"
)

func TestWeightedLength(t *testing.T) {
	tests := []struct {
		name string
		text string
		want int
	}{
		{name: "ascii", text: "hello", want: 5},
		{name: "empty", text: "", want: 0},
		{name: "cjk doubles", text: "你好", want: 4},    // 2 CJK runes x2
		{name: "emoji doubles", text: "hi👍", want: 4}, // 2 + 2
		{name: "url fixed 23", text: "see https://example.com/very/long/path/that/is/way/longer/than/23", want: 4 + 23}, // "see " = 4
		{name: "www url", text: "www.example.com", want: 23},
		{name: "mixed ascii+cjk", text: "ab你", want: 1 + 1 + 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := weightedLength(tt.text); got != tt.want {
				t.Errorf("weightedLength(%q) = %d, want %d", tt.text, got, tt.want)
			}
		})
	}
}

func TestWeightedLength_LimitBoundary(t *testing.T) {
	// 280 ASCII chars == exactly the limit.
	at := strings.Repeat("a", maxWeightedLen)
	if got := weightedLength(at); got != maxWeightedLen {
		t.Errorf("280 ascii = %d, want %d", got, maxWeightedLen)
	}
	// 140 CJK runes == 280 weighted (also exactly the limit).
	cjk := strings.Repeat("好", 140)
	if got := weightedLength(cjk); got != maxWeightedLen {
		t.Errorf("140 cjk = %d, want %d", got, maxWeightedLen)
	}
	// One more ASCII char tips over.
	if got := weightedLength(at + "a"); got <= maxWeightedLen {
		t.Errorf("281 ascii = %d, want > %d", got, maxWeightedLen)
	}
}
