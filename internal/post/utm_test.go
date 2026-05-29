package post

import (
	"strings"
	"testing"
)

func TestApplyUTM(t *testing.T) {
	params := map[string]string{"utm_source": "postal", "utm_medium": "social"}

	tests := []struct {
		name     string
		text     string
		contains []string
		exact    string // when set, output must equal this
	}{
		{name: "adds params to url", text: "check https://example.com/page", contains: []string{"utm_source=postal", "utm_medium=social", "https://example.com/page?"}},
		{name: "preserves existing query", text: "https://example.com/p?a=1", contains: []string{"a=1", "utm_source=postal"}},
		{name: "no urls unchanged", text: "just some text, no links", exact: "just some text, no links"},
		{name: "trailing period preserved outside url", text: "see https://example.com/p.", contains: []string{"https://example.com/p?", "utm_source=postal", "/p?utm"}},
		{name: "wrapping parens not absorbed", text: "(https://example.com)", contains: []string{"https://example.com?", "utm_source=postal", ")"}},
		{name: "multiple urls both tagged", text: "https://a.com and https://b.com", contains: []string{"a.com?", "b.com?", "utm_source=postal", "utm_medium=social"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ApplyUTM(tt.text, params)
			if tt.exact != "" && got != tt.exact {
				t.Fatalf("ApplyUTM = %q, want %q", got, tt.exact)
			}
			for _, want := range tt.contains {
				if !strings.Contains(got, want) {
					t.Errorf("ApplyUTM(%q) = %q, missing %q", tt.text, got, want)
				}
			}
		})
	}
}

func TestApplyUTM_NoParamsUnchanged(t *testing.T) {
	text := "https://example.com/page"
	if got := ApplyUTM(text, nil); got != text {
		t.Errorf("ApplyUTM with no params = %q, want unchanged", got)
	}
}
