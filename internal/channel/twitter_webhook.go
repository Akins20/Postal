package channel

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
)

// maxWebhookBody bounds an inbound event body to defeat memory-exhaustion.
const maxWebhookBody = 1 << 20 // 1 MiB

// TwitterWebhookHandler implements X's Account Activity webhook contract: the
// CRC GET challenge that proves endpoint ownership, and signed POST event
// delivery. Both use HMAC-SHA256 keyed by the app's consumer secret.
type TwitterWebhookHandler struct {
	secret []byte
	log    *slog.Logger
}

// NewTwitterWebhookHandler builds the handler from the app consumer secret.
func NewTwitterWebhookHandler(secret string, log *slog.Logger) *TwitterWebhookHandler {
	if log == nil {
		log = slog.Default()
	}
	return &TwitterWebhookHandler{secret: []byte(secret), log: log}
}

// RegisterPublic mounts the webhook routes. They are public by design: X
// authenticates via the request signature, not a session.
func (h *TwitterWebhookHandler) RegisterPublic(r chi.Router) {
	r.Get("/webhooks/twitter", h.crc)
	r.Post("/webhooks/twitter", h.event)
}

// crc answers the Challenge-Response Check: echo back a base64 HMAC-SHA256 of
// the crc_token, keyed by the consumer secret, in X's expected envelope.
func (h *TwitterWebhookHandler) crc(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("crc_token")
	if token == "" {
		http.Error(w, `{"error":"missing crc_token"}`, http.StatusBadRequest)
		return
	}
	mac := hmac.New(sha256.New, h.secret)
	mac.Write([]byte(token))
	resp := "sha256=" + base64.StdEncoding.EncodeToString(mac.Sum(nil))
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"response_token": resp})
}

// event receives an activity payload. It verifies the x-twitter-webhooks-
// signature header (base64 HMAC-SHA256 of the raw body) before acknowledging.
// Postal does not yet act on these events, so a verified payload is logged and
// acknowledged with 200 so X does not retry.
func (h *TwitterWebhookHandler) event(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, maxWebhookBody))
	if err != nil {
		http.Error(w, `{"error":"unreadable body"}`, http.StatusBadRequest)
		return
	}
	mac := hmac.New(sha256.New, h.secret)
	mac.Write(body)
	expected := "sha256=" + base64.StdEncoding.EncodeToString(mac.Sum(nil))
	got := r.Header.Get("x-twitter-webhooks-signature")
	if got == "" || !hmac.Equal([]byte(got), []byte(expected)) {
		h.log.WarnContext(r.Context(), "twitter webhook signature mismatch")
		http.Error(w, `{"error":"invalid signature"}`, http.StatusUnauthorized)
		return
	}
	h.log.InfoContext(r.Context(), "twitter webhook event received", slog.Int("bytes", len(body)))
	w.WriteHeader(http.StatusOK)
}
