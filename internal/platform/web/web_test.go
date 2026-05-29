package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Akins20/postal/internal/platform/apperr"
)

func TestRespond(t *testing.T) {
	rec := httptest.NewRecorder()
	Respond(rec, http.StatusCreated, map[string]string{"id": "abc"})

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusCreated)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		t.Errorf("content-type = %q", ct)
	}
	var env Envelope
	if err := json.NewDecoder(rec.Body).Decode(&env); err != nil {
		t.Fatalf("decode: %v", err)
	}
	data, _ := env.Data.(map[string]any)
	if data["id"] != "abc" {
		t.Errorf("data.id = %v, want abc", data["id"])
	}
}

func TestFail_MapsKindToStatus(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   string
	}{
		{"validation", apperr.Validation("bad_input", "nope"), http.StatusBadRequest, "bad_input"},
		{"unauthorized", apperr.Unauthorized("no_token", "login required"), http.StatusUnauthorized, "no_token"},
		{"forbidden", apperr.Forbidden("denied", "not allowed"), http.StatusForbidden, "denied"},
		{"not found", apperr.NotFound("missing", "gone"), http.StatusNotFound, "missing"},
		{"conflict", apperr.Conflict("dup", "exists"), http.StatusConflict, "dup"},
		{"rate limited", apperr.RateLimited("slow_down", "too many"), http.StatusTooManyRequests, "slow_down"},
		{"plain error is internal", errInternal(), http.StatusInternalServerError, "internal_error"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			rec := httptest.NewRecorder()

			Fail(rec, req, nil, tt.err)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			var env ErrorEnvelope
			if err := json.NewDecoder(rec.Body).Decode(&env); err != nil {
				t.Fatalf("decode: %v", err)
			}
			if env.Error.Code != tt.wantCode {
				t.Errorf("code = %q, want %q", env.Error.Code, tt.wantCode)
			}
		})
	}
}

func TestFail_InternalDoesNotLeakCause(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	Fail(rec, req, nil, apperr.Internal(errSecret()))

	body := rec.Body.String()
	if strings.Contains(body, "super-secret-detail") {
		t.Errorf("internal cause leaked to client: %s", body)
	}
}

func TestHandler_RendersReturnedError(t *testing.T) {
	h := Handler(nil, func(_ http.ResponseWriter, _ *http.Request) error {
		return apperr.NotFound("missing", "no such thing")
	})
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	h(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestHandler_NoErrorLeavesResponseUntouched(t *testing.T) {
	h := Handler(nil, func(w http.ResponseWriter, _ *http.Request) error {
		Respond(w, http.StatusOK, "ok")
		return nil
	})
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	h(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestDecodeJSON(t *testing.T) {
	type payload struct {
		Name string `json:"name"`
	}

	tests := []struct {
		name      string
		body      string
		ctype     string
		wantErr   bool
		wantField string
	}{
		{name: "valid", body: `{"name":"postal"}`, ctype: "application/json", wantErr: false},
		{name: "unknown field", body: `{"name":"x","extra":1}`, ctype: "application/json", wantErr: true},
		{name: "malformed", body: `{"name":`, ctype: "application/json", wantErr: true},
		{name: "empty", body: ``, ctype: "application/json", wantErr: true},
		{name: "trailing data", body: `{"name":"x"}{"name":"y"}`, ctype: "application/json", wantErr: true},
		{name: "wrong content type", body: `{"name":"x"}`, ctype: "text/plain", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", tt.ctype)
			rec := httptest.NewRecorder()

			var dst payload
			err := DecodeJSON(rec, req, &dst)
			if tt.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !tt.wantErr && dst.Name != "postal" {
				t.Errorf("decoded name = %q", dst.Name)
			}
		})
	}
}

func TestDecodeJSONLimit_RejectsOversize(t *testing.T) {
	big := `{"name":"` + strings.Repeat("a", 100) + `"}`
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(big))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	var dst struct {
		Name string `json:"name"`
	}
	err := DecodeJSONLimit(rec, req, &dst, 16)
	if err == nil {
		t.Fatal("expected size-limit error, got nil")
	}
	if apperr.KindOf(err) != apperr.KindValidation {
		t.Errorf("kind = %v, want validation", apperr.KindOf(err))
	}
}

// errInternal and errSecret are tiny helpers returning plain (non-apperr) errors.
func errInternal() error { return &simpleErr{"boom"} }
func errSecret() error   { return &simpleErr{"super-secret-detail"} }

type simpleErr struct{ msg string }

func (e *simpleErr) Error() string { return e.msg }
