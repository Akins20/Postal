package channel

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
)

const testWebhookSecret = "test-consumer-secret"

func newWebhookRouter() chi.Router {
	r := chi.NewRouter()
	NewTwitterWebhookHandler(testWebhookSecret, nil).RegisterPublic(r)
	return r
}

func TestTwitterWebhookCRC(t *testing.T) {
	t.Parallel()
	r := newWebhookRouter()

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/webhooks/twitter?crc_token=abc123", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("crc status = %d, want 200", rec.Code)
	}

	var body struct {
		ResponseToken string `json:"response_token"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("crc body not JSON: %v", err)
	}
	// The response must equal sha256=<base64(HMAC-SHA256(crc_token, secret))>.
	mac := hmac.New(sha256.New, []byte(testWebhookSecret))
	mac.Write([]byte("abc123"))
	want := "sha256=" + base64.StdEncoding.EncodeToString(mac.Sum(nil))
	if body.ResponseToken != want {
		t.Fatalf("response_token = %q, want %q", body.ResponseToken, want)
	}
	if !strings.HasPrefix(body.ResponseToken, "sha256=") {
		t.Fatalf("response_token missing sha256= prefix: %q", body.ResponseToken)
	}
}

func TestTwitterWebhookCRCMissingToken(t *testing.T) {
	t.Parallel()
	rec := httptest.NewRecorder()
	newWebhookRouter().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/webhooks/twitter", nil))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("missing crc_token status = %d, want 400", rec.Code)
	}
}

func TestTwitterWebhookEventSignature(t *testing.T) {
	t.Parallel()
	r := newWebhookRouter()
	payload := `{"for_user_id":"123"}`

	// Correct signature -> 200.
	mac := hmac.New(sha256.New, []byte(testWebhookSecret))
	mac.Write([]byte(payload))
	sig := "sha256=" + base64.StdEncoding.EncodeToString(mac.Sum(nil))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/webhooks/twitter", strings.NewReader(payload))
	req.Header.Set("x-twitter-webhooks-signature", sig)
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("valid signature status = %d, want 200", rec.Code)
	}

	// Wrong signature -> 401.
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/webhooks/twitter", strings.NewReader(payload))
	req.Header.Set("x-twitter-webhooks-signature", "sha256=wrong")
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("bad signature status = %d, want 401", rec.Code)
	}
}
