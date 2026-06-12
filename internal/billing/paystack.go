package billing

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"
)

// defaultPaystackAPIBase is Paystack's live API host (overridable for tests).
const defaultPaystackAPIBase = "https://api.paystack.co"

// PaystackProvider initializes Paystack transactions (cards/bank across
// Africa, charged in NGN) and verifies its webhooks.
type PaystackProvider struct {
	secretKey string
	apiBase   string
	ngnPerUSD int64
	http      *http.Client
}

// NewPaystackProvider builds the Paystack provider. apiBase "" means the real
// Paystack API.
func NewPaystackProvider(secretKey, apiBase string, ngnPerUSD int64, client *http.Client) *PaystackProvider {
	if apiBase == "" {
		apiBase = defaultPaystackAPIBase
	}
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	if ngnPerUSD <= 0 {
		ngnPerUSD = 1
	}
	return &PaystackProvider{secretKey: secretKey, apiBase: apiBase, ngnPerUSD: ngnPerUSD, http: client}
}

// Name implements Provider.
func (p *PaystackProvider) Name() string { return "paystack" }

// CreateCheckout initializes a Paystack transaction (amount in kobo, converted
// from the USD price at the configured rate) and returns its authorization
// URL. workspace_id and credits ride in metadata for the webhook.
func (p *PaystackProvider) CreateCheckout(ctx context.Context, in CheckoutInput) (string, error) {
	koboAmount := in.USDCents * p.ngnPerUSD // cents * (NGN/USD) = kobo
	payload := map[string]any{
		"email":        fmt.Sprintf("workspace-%s@postal.invalid", in.WorkspaceID), // Paystack requires an email
		"amount":       koboAmount,
		"currency":     "NGN",
		"callback_url": in.ReturnURL + "?status=success",
		"metadata": map[string]string{
			"workspace_id": in.WorkspaceID.String(),
			"credits":      strconv.FormatInt(in.Credits, 10),
		},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("encoding paystack request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		p.apiBase+"/transaction/initialize", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("building paystack request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+p.secretKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("calling paystack: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("paystack initialize returned %d", resp.StatusCode)
	}
	var out struct {
		Status bool `json:"status"`
		Data   struct {
			AuthorizationURL string `json:"authorization_url"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &out); err != nil || !out.Status || out.Data.AuthorizationURL == "" {
		return "", fmt.Errorf("parsing paystack initialize response: %w", err)
	}
	return out.Data.AuthorizationURL, nil
}

// VerifyWebhook checks the x-paystack-signature header (HMAC-SHA512 of the raw
// body with the secret key) and extracts a top-up from charge.success events.
// Other event types return (nil, nil).
func (p *PaystackProvider) VerifyWebhook(payload []byte, sigHeader string) (*TopupEvent, error) {
	mac := hmac.New(sha512.New, []byte(p.secretKey))
	mac.Write(payload)
	expected := hex.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(sigHeader), []byte(expected)) {
		return nil, fmt.Errorf("paystack webhook signature mismatch")
	}

	var evt struct {
		Event string `json:"event"`
		Data  struct {
			Reference string `json:"reference"`
			Status    string `json:"status"`
			Metadata  struct {
				WorkspaceID string `json:"workspace_id"`
				Credits     string `json:"credits"`
			} `json:"metadata"`
		} `json:"data"`
	}
	if err := json.Unmarshal(payload, &evt); err != nil {
		return nil, fmt.Errorf("parsing paystack event: %w", err)
	}
	if evt.Event != "charge.success" || evt.Data.Status != "success" {
		return nil, nil // acknowledge and ignore
	}
	credits, err := strconv.ParseInt(evt.Data.Metadata.Credits, 10, 64)
	if err != nil || credits <= 0 {
		return nil, fmt.Errorf("paystack event missing credits metadata")
	}
	if evt.Data.Metadata.WorkspaceID == "" || evt.Data.Reference == "" {
		return nil, fmt.Errorf("paystack event missing workspace/reference")
	}
	return &TopupEvent{
		Reference:   "paystack:" + evt.Data.Reference,
		WorkspaceID: evt.Data.Metadata.WorkspaceID,
		Credits:     credits,
	}, nil
}
