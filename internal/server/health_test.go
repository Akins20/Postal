package server

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

// stubPinger is a test double for a dependency health check.
type stubPinger struct {
	err error
}

func (s stubPinger) Ping(context.Context) error { return s.err }

func TestHandleHealthz(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()

	handleHealthz(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	var body map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decoding body: %v", err)
	}
	if body["status"] != "ok" {
		t.Errorf("status = %q, want %q", body["status"], "ok")
	}
}

func TestHandleReadyz(t *testing.T) {
	tests := []struct {
		name       string
		db, cache  Pinger
		wantStatus int
		wantState  string
	}{
		{
			name:       "all healthy",
			db:         stubPinger{},
			cache:      stubPinger{},
			wantStatus: http.StatusOK,
			wantState:  "ready",
		},
		{
			name:       "database down",
			db:         stubPinger{err: errors.New("connection refused")},
			cache:      stubPinger{},
			wantStatus: http.StatusServiceUnavailable,
			wantState:  "not ready",
		},
		{
			name:       "redis down",
			db:         stubPinger{},
			cache:      stubPinger{err: errors.New("timeout")},
			wantStatus: http.StatusServiceUnavailable,
			wantState:  "not ready",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
			rec := httptest.NewRecorder()

			handleReadyz(tt.db, tt.cache)(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			var body map[string]any
			if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
				t.Fatalf("decoding body: %v", err)
			}
			if body["status"] != tt.wantState {
				t.Errorf("status = %v, want %q", body["status"], tt.wantState)
			}
		})
	}
}
