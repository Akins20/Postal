package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func ok(http.ResponseWriter, *http.Request) {}

func TestSecurityHeaders(t *testing.T) {
	t.Run("production sets HSTS + hardening headers", func(t *testing.T) {
		h := securityHeaders(true)(http.HandlerFunc(ok))
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/x", nil))
		want := map[string]string{
			"X-Content-Type-Options":    "nosniff",
			"X-Frame-Options":           "DENY",
			"Referrer-Policy":           "no-referrer",
			"Content-Security-Policy":   "default-src 'none'; frame-ancestors 'none'",
			"Strict-Transport-Security": hstsMaxAge,
		}
		for k, v := range want {
			if got := rec.Header().Get(k); got != v {
				t.Errorf("header %s = %q, want %q", k, got, v)
			}
		}
	})

	t.Run("non-production omits HSTS", func(t *testing.T) {
		h := securityHeaders(false)(http.HandlerFunc(ok))
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/x", nil))
		if got := rec.Header().Get("Strict-Transport-Security"); got != "" {
			t.Errorf("HSTS set in dev: %q", got)
		}
		if got := rec.Header().Get("X-Frame-Options"); got != "DENY" {
			t.Errorf("X-Frame-Options = %q, want DENY", got)
		}
	})
}

func TestCORS(t *testing.T) {
	mw := cors([]string{"https://app.postal.test"})

	t.Run("allowed origin is reflected with credentials", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/x", nil)
		req.Header.Set("Origin", "https://app.postal.test")
		mw(http.HandlerFunc(ok)).ServeHTTP(rec, req)
		if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "https://app.postal.test" {
			t.Errorf("ACAO = %q, want the allowed origin", got)
		}
		if got := rec.Header().Get("Access-Control-Allow-Credentials"); got != "true" {
			t.Errorf("ACAC = %q, want true", got)
		}
	})

	t.Run("disallowed origin is not reflected", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/x", nil)
		req.Header.Set("Origin", "https://evil.example")
		mw(http.HandlerFunc(ok)).ServeHTTP(rec, req)
		if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
			t.Errorf("ACAO leaked for disallowed origin: %q", got)
		}
	})

	t.Run("preflight from allowed origin returns 204 with methods", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodOptions, "/x", nil)
		req.Header.Set("Origin", "https://app.postal.test")
		called := false
		mw(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { called = true })).ServeHTTP(rec, req)
		if rec.Code != http.StatusNoContent {
			t.Errorf("preflight status = %d, want 204", rec.Code)
		}
		if called {
			t.Error("preflight should short-circuit, not reach the handler")
		}
		if rec.Header().Get("Access-Control-Allow-Methods") == "" {
			t.Error("preflight missing Access-Control-Allow-Methods")
		}
	})

	t.Run("never reflects '*' (credentials safety)", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/x", nil)
		req.Header.Set("Origin", "https://app.postal.test")
		mw(http.HandlerFunc(ok)).ServeHTTP(rec, req)
		if rec.Header().Get("Access-Control-Allow-Origin") == "*" {
			t.Error("wildcard origin must never be used with credentials")
		}
	})
}
