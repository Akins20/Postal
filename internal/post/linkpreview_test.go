package post

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLinkPreviewFetchParsesOpenGraph(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(`<html><head>
			<title>Fallback title</title>
			<meta property="og:title" content="Launch day" />
			<meta content="The big announcement" property="og:description"/>
			<meta property='og:image' content='https://cdn.example/x.png'>
			<meta property="og:site_name" content="Example">
		</head><body>hi</body></html>`))
	}))
	defer ts.Close()

	pv, err := newTestLinkPreviewer().Fetch(context.Background(), ts.URL)
	require.NoError(t, err)
	assert.Equal(t, "Launch day", pv.Title)
	assert.Equal(t, "The big announcement", pv.Description)
	assert.Equal(t, "https://cdn.example/x.png", pv.Image)
	assert.Equal(t, "Example", pv.SiteName)
}

func TestLinkPreviewFallsBackToTitleTag(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`<html><head><title> Plain page </title></head></html>`))
	}))
	defer ts.Close()

	pv, err := newTestLinkPreviewer().Fetch(context.Background(), ts.URL)
	require.NoError(t, err)
	assert.Equal(t, "Plain page", pv.Title)
}

func TestLinkPreviewRejectsBadInput(t *testing.T) {
	p := newTestLinkPreviewer()
	for _, raw := range []string{"", "ftp://example.com/x", "javascript:alert(1)", "not a url"} {
		_, err := p.Fetch(context.Background(), raw)
		assert.Error(t, err, raw)
	}
}

func TestLinkPreviewBlocksPrivateAddresses(t *testing.T) {
	// Production guard: loopback must be refused at connect time.
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("secret"))
	}))
	defer ts.Close()

	_, err := NewLinkPreviewer().Fetch(context.Background(), ts.URL)
	require.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "publicly routable") ||
		strings.Contains(err.Error(), "fetching link"), err.Error())
}

func TestLinkPreviewClampsHostileFields(t *testing.T) {
	long := strings.Repeat("a", 5000)
	pv := parseOpenGraph(`<meta property="og:title" content="` + long + `">`)
	assert.Len(t, pv.Title, linkPreviewMaxField)
}
