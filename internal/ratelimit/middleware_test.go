package ratelimit

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Akins20/postal/internal/platform/web"
)

func TestMiddleware_Returns429PastThreshold(t *testing.T) {
	rdb := testRedis(t)
	defer func() { _ = rdb.Close() }()

	ctx := context.Background()
	now := time.UnixMilli(2_000_000)
	lim := NewLimiter(rdb, func() time.Time { return now })

	const capacity = 3
	cfg := Config{
		Rule:   Rule{Capacity: capacity, RefillRate: 0}, // no refill within the test window
		Prefix: "test:mw:" + t.Name(),
		Key:    func(_ *http.Request) string { return "fixed-client" },
	}
	t.Cleanup(func() { rdb.Del(ctx, cfg.Prefix+":fixed-client") })

	handler := lim.Middleware(cfg)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		web.Respond(w, http.StatusOK, "pong")
	}))

	// First `capacity` requests pass.
	for i := 0; i < capacity; i++ {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/ping", nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("request #%d: status = %d, want 200", i, rec.Code)
		}
		if rec.Header().Get("X-RateLimit-Limit") != "3" {
			t.Errorf("missing/incorrect X-RateLimit-Limit header: %q", rec.Header().Get("X-RateLimit-Limit"))
		}
	}

	// Next request is rejected with the standard error envelope.
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/ping", nil))
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("over-threshold status = %d, want 429", rec.Code)
	}
	var env web.ErrorEnvelope
	if err := json.NewDecoder(rec.Body).Decode(&env); err != nil {
		t.Fatalf("decode error envelope: %v", err)
	}
	if env.Error.Code != "rate_limited" {
		t.Errorf("error code = %q, want rate_limited", env.Error.Code)
	}
}

func TestMiddleware_FailClosedOnBackendError(t *testing.T) {
	// A limiter pointed at a dead Redis fails closed by default (429), protecting
	// the endpoint when the backend is unavailable.
	dead := deadScripter{}
	lim := NewLimiter(dead, func() time.Time { return time.UnixMilli(0) })

	handler := lim.Middleware(Config{
		Rule:   Rule{Capacity: 1, RefillRate: 1},
		Prefix: "test:failclosed",
	})(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		web.Respond(w, http.StatusOK, "pong")
	}))

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusTooManyRequests {
		t.Errorf("fail-closed status = %d, want 429", rec.Code)
	}
}

func TestMiddleware_FailOpenOnBackendError(t *testing.T) {
	dead := deadScripter{}
	lim := NewLimiter(dead, func() time.Time { return time.UnixMilli(0) })

	handler := lim.Middleware(Config{
		Rule:     Rule{Capacity: 1, RefillRate: 1},
		Prefix:   "test:failopen",
		FailOpen: true,
	})(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		web.Respond(w, http.StatusOK, "pong")
	}))

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusOK {
		t.Errorf("fail-open status = %d, want 200", rec.Code)
	}
}
