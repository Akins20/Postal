package billing

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// defaultStripeAPIBase is Stripe's live API host (overridable for tests).
const defaultStripeAPIBase = "https://api.stripe.com"

// stripeTolerance bounds webhook timestamp age (replay protection).
const stripeTolerance = 5 * time.Minute

// StripeProvider creates Stripe Checkout Sessions and verifies its webhooks.
type StripeProvider struct {
	secretKey     string
	webhookSecret string
	apiBase       string
	http          *http.Client
	now           func() time.Time
}

// NewStripeProvider builds the Stripe provider. apiBase "" means the real
// Stripe API; now nil means time.Now.
func NewStripeProvider(secretKey, webhookSecret, apiBase string, client *http.Client, now func() time.Time) *StripeProvider {
	if apiBase == "" {
		apiBase = defaultStripeAPIBase
	}
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	if now == nil {
		now = time.Now
	}
	return &StripeProvider{secretKey: secretKey, webhookSecret: webhookSecret, apiBase: apiBase, http: client, now: now}
}

// Name implements Provider.
func (p *StripeProvider) Name() string { return "stripe" }

// CreateCheckout creates a Checkout Session for the credit pack and returns
// its hosted URL. workspace_id and credits ride in metadata; the webhook reads
// them back when the payment completes.
func (p *StripeProvider) CreateCheckout(ctx context.Context, in CheckoutInput) (string, error) {
	form := url.Values{}
	form.Set("mode", "payment")
	form.Set("success_url", in.ReturnURL+"?status=success")
	form.Set("cancel_url", in.ReturnURL+"?status=canceled")
	form.Set("line_items[0][quantity]", "1")
	form.Set("line_items[0][price_data][currency]", "usd")
	form.Set("line_items[0][price_data][unit_amount]", strconv.FormatInt(in.USDCents, 10))
	form.Set("line_items[0][price_data][product_data][name]",
		fmt.Sprintf("Postal wallet top-up (%d credits)", in.Credits))
	form.Set("metadata[workspace_id]", in.WorkspaceID.String())
	form.Set("metadata[credits]", strconv.FormatInt(in.Credits, 10))

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		p.apiBase+"/v1/checkout/sessions", strings.NewReader(form.Encode()))
	if err != nil {
		return "", fmt.Errorf("building stripe request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+p.secretKey)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := p.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("calling stripe: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("stripe checkout returned %d", resp.StatusCode)
	}
	var out struct {
		URL string `json:"url"`
	}
	if err := json.Unmarshal(body, &out); err != nil || out.URL == "" {
		return "", fmt.Errorf("parsing stripe checkout response: %w", err)
	}
	return out.URL, nil
}

// TopupEvent is a verified, parsed "payment completed" webhook from either
// provider, carrying what the wallet credit needs.
type TopupEvent struct {
	// Reference uniquely identifies the payment (ledger idempotency key).
	Reference   string
	WorkspaceID string
	Credits     int64
}

// VerifyWebhook checks the Stripe-Signature header (HMAC-SHA256 over
// "<t>.<payload>", within tolerance) and extracts a top-up from
// checkout.session.completed events. Other event types return (nil, nil).
func (p *StripeProvider) VerifyWebhook(payload []byte, sigHeader string) (*TopupEvent, error) {
	ts, sigs, err := parseStripeSigHeader(sigHeader)
	if err != nil {
		return nil, err
	}
	age := p.now().Unix() - ts
	if age < 0 {
		age = -age
	}
	if age > int64(stripeTolerance.Seconds()) {
		return nil, fmt.Errorf("stripe webhook timestamp outside tolerance")
	}
	mac := hmac.New(sha256.New, []byte(p.webhookSecret))
	mac.Write([]byte(strconv.FormatInt(ts, 10)))
	mac.Write([]byte("."))
	mac.Write(payload)
	expected := hex.EncodeToString(mac.Sum(nil))
	ok := false
	for _, s := range sigs {
		if hmac.Equal([]byte(s), []byte(expected)) {
			ok = true
			break
		}
	}
	if !ok {
		return nil, fmt.Errorf("stripe webhook signature mismatch")
	}

	var evt struct {
		ID   string `json:"id"`
		Type string `json:"type"`
		Data struct {
			Object struct {
				PaymentStatus string            `json:"payment_status"`
				Metadata      map[string]string `json:"metadata"`
			} `json:"object"`
		} `json:"data"`
	}
	if err := json.Unmarshal(payload, &evt); err != nil {
		return nil, fmt.Errorf("parsing stripe event: %w", err)
	}
	if evt.Type != "checkout.session.completed" || evt.Data.Object.PaymentStatus != "paid" {
		return nil, nil // not a top-up completion; acknowledge and ignore
	}
	credits, err := strconv.ParseInt(evt.Data.Object.Metadata["credits"], 10, 64)
	if err != nil || credits <= 0 {
		return nil, fmt.Errorf("stripe event missing credits metadata")
	}
	ws := evt.Data.Object.Metadata["workspace_id"]
	if ws == "" {
		return nil, fmt.Errorf("stripe event missing workspace_id metadata")
	}
	return &TopupEvent{Reference: "stripe:" + evt.ID, WorkspaceID: ws, Credits: credits}, nil
}

// parseStripeSigHeader splits "t=...,v1=...,v1=..." into timestamp + v1 sigs.
func parseStripeSigHeader(h string) (ts int64, sigs []string, err error) {
	for _, part := range strings.Split(h, ",") {
		k, v, found := strings.Cut(strings.TrimSpace(part), "=")
		if !found {
			continue
		}
		switch k {
		case "t":
			ts, err = strconv.ParseInt(v, 10, 64)
			if err != nil {
				return 0, nil, fmt.Errorf("bad stripe signature timestamp")
			}
		case "v1":
			sigs = append(sigs, v)
		}
	}
	if ts == 0 || len(sigs) == 0 {
		return 0, nil, fmt.Errorf("malformed stripe signature header")
	}
	return ts, sigs, nil
}
