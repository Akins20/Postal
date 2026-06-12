package integration

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stubOGShortener mimics ogshortener.site's API shape (X-API-Key auth,
// POST /api/v1/links -> 201 {shortUrl}).
func stubOGShortener(t *testing.T, key string) (*httptest.Server, *int) {
	t.Helper()
	calls := 0
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/links", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-API-Key") != key {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"success": true, "data": []any{}})
	})
	mux.HandleFunc("POST /api/v1/links", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-API-Key") != key {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		calls++
		var in struct {
			URL string `json:"url"`
		}
		_ = json.NewDecoder(r.Body).Decode(&in)
		if in.URL == "" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]string{"shortUrl": "https://ogsh.rt/abc" + string(rune('0'+calls))})
	})
	ts := httptest.NewServer(mux)
	t.Cleanup(ts.Close)
	return ts, &calls
}

func TestOGShortenerVerify(t *testing.T) {
	ts, _ := stubOGShortener(t, "ogl_good")
	c := NewOGShortenerClient(ts.URL, nil)
	require.NoError(t, c.Verify(context.Background(), "ogl_good"))
	assert.Error(t, c.Verify(context.Background(), "ogl_wrong"))
}

func TestShortenTextReplacesEveryLinkOnce(t *testing.T) {
	ts, calls := stubOGShortener(t, "ogl_good")
	c := NewOGShortenerClient(ts.URL, nil)

	in := "Read https://example.com/a then https://example.com/b and again https://example.com/a"
	out, err := c.ShortenText(context.Background(), "ogl_good", in)
	require.NoError(t, err)
	assert.NotContains(t, out, "example.com")
	assert.Contains(t, out, "https://ogsh.rt/")
	assert.Equal(t, 2, *calls, "duplicate URLs shorten once")
}

func TestShortenTextNoLinksIsNoop(t *testing.T) {
	c := NewOGShortenerClient("http://unreachable.invalid", nil)
	out, err := c.ShortenText(context.Background(), "ogl_x", "no links here")
	require.NoError(t, err)
	assert.Equal(t, "no links here", out)
}

func TestShortenTextAbortsOnProviderError(t *testing.T) {
	ts, _ := stubOGShortener(t, "ogl_good")
	c := NewOGShortenerClient(ts.URL, nil)
	_, err := c.ShortenText(context.Background(), "ogl_bad", "see https://example.com/a")
	assert.Error(t, err, "bad key must abort, not half-rewrite")
}
