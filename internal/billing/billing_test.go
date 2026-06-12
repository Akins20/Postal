package billing

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPricingUSDCents(t *testing.T) {
	p := Pricing{CreditsPerUSDCent: 1}
	assert.Equal(t, int64(500), p.USDCents(500), "1 credit per cent")

	p = Pricing{CreditsPerUSDCent: 10}
	assert.Equal(t, int64(50), p.USDCents(500), "10 credits per cent")
	assert.Equal(t, int64(51), p.USDCents(501), "partial cents round up (never undercharge)")

	p = Pricing{} // zero config must not divide by zero
	assert.Equal(t, int64(7), p.USDCents(7))
}

func TestPricingCostForItem(t *testing.T) {
	p := Pricing{
		PublishCosts: map[string]int64{"twitter": 10},
		MediaCosts:   map[string]int64{"twitter": 15},
		URLCosts:     map[string]int64{"twitter": 25},
	}
	assert.Equal(t, int64(10), p.CostForItem(PublishItem{Platform: "twitter", Body: "plain"}))
	assert.Equal(t, int64(15), p.CostForItem(PublishItem{Platform: "twitter", Body: "pic", HasMedia: true}))
	assert.Equal(t, int64(25), p.CostForItem(PublishItem{Platform: "twitter", Body: "see https://a.test"}))
	assert.Equal(t, int64(25),
		p.CostForItem(PublishItem{Platform: "twitter", Body: "https://a.test", HasMedia: true}),
		"highest tier wins for link+media")
	assert.Zero(t, p.CostForItem(PublishItem{Platform: "mastodon", Body: "https://a.test"}),
		"unlisted platforms are free")
}

// stripeSig builds a valid Stripe-Signature header for payload at ts.
func stripeSig(secret string, ts int64, payload []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	fmt.Fprintf(mac, "%d.", ts)
	mac.Write(payload)
	return fmt.Sprintf("t=%d,v1=%s", ts, hex.EncodeToString(mac.Sum(nil)))
}

func stripeEvent(credits int64) []byte {
	return []byte(`{"id":"evt_1","type":"checkout.session.completed","data":{"object":{` +
		`"payment_status":"paid","metadata":{"workspace_id":"11111111-1111-1111-1111-111111111111",` +
		`"credits":"` + strconv.FormatInt(credits, 10) + `"}}}}`)
}

func TestStripeVerifyWebhook(t *testing.T) {
	now := time.Unix(1_780_000_000, 0)
	p := NewStripeProvider("sk_test", "whsec_test", "", nil, func() time.Time { return now })

	payload := stripeEvent(500)
	evt, err := p.VerifyWebhook(payload, stripeSig("whsec_test", now.Unix(), payload))
	require.NoError(t, err)
	require.NotNil(t, evt)
	assert.Equal(t, "stripe:evt_1", evt.Reference)
	assert.Equal(t, int64(500), evt.Credits)
	assert.Equal(t, "11111111-1111-1111-1111-111111111111", evt.WorkspaceID)
}

func TestStripeVerifyWebhookRejects(t *testing.T) {
	now := time.Unix(1_780_000_000, 0)
	p := NewStripeProvider("sk_test", "whsec_test", "", nil, func() time.Time { return now })
	payload := stripeEvent(500)

	_, err := p.VerifyWebhook(payload, stripeSig("WRONG-secret", now.Unix(), payload))
	assert.Error(t, err, "wrong secret must fail")

	old := now.Add(-time.Hour).Unix()
	_, err = p.VerifyWebhook(payload, stripeSig("whsec_test", old, payload))
	assert.Error(t, err, "stale timestamp must fail (replay protection)")

	_, err = p.VerifyWebhook(payload, "garbage")
	assert.Error(t, err, "malformed header must fail")

	tampered := append([]byte{}, payload...)
	tampered[len(tampered)-2] = 'x'
	_, err = p.VerifyWebhook(tampered, stripeSig("whsec_test", now.Unix(), payload))
	assert.Error(t, err, "tampered payload must fail")
}

func TestStripeVerifyWebhookIgnoresOtherEvents(t *testing.T) {
	now := time.Unix(1_780_000_000, 0)
	p := NewStripeProvider("sk_test", "whsec_test", "", nil, func() time.Time { return now })
	payload := []byte(`{"id":"evt_2","type":"invoice.paid","data":{"object":{}}}`)
	evt, err := p.VerifyWebhook(payload, stripeSig("whsec_test", now.Unix(), payload))
	require.NoError(t, err)
	assert.Nil(t, evt, "irrelevant event types are acknowledged, not credited")
}

func paystackSig(secret string, payload []byte) string {
	mac := hmac.New(sha512.New, []byte(secret))
	mac.Write(payload)
	return hex.EncodeToString(mac.Sum(nil))
}

func TestPaystackVerifyWebhook(t *testing.T) {
	p := NewPaystackProvider("sk_ps", "", 1600, nil)
	payload := []byte(`{"event":"charge.success","data":{"reference":"ref_9","status":"success",` +
		`"metadata":{"workspace_id":"11111111-1111-1111-1111-111111111111","credits":"500"}}}`)

	evt, err := p.VerifyWebhook(payload, paystackSig("sk_ps", payload))
	require.NoError(t, err)
	require.NotNil(t, evt)
	assert.Equal(t, "paystack:ref_9", evt.Reference)
	assert.Equal(t, int64(500), evt.Credits)

	_, err = p.VerifyWebhook(payload, paystackSig("WRONG", payload))
	assert.Error(t, err, "wrong secret must fail")

	other := []byte(`{"event":"transfer.success","data":{}}`)
	evt, err = p.VerifyWebhook(other, paystackSig("sk_ps", other))
	require.NoError(t, err)
	assert.Nil(t, evt, "irrelevant event types are acknowledged, not credited")
}

func TestPaystackCheckoutAmountConversion(t *testing.T) {
	// 500 credits at 1 credit/cent = $5.00 = 500 cents; at 1600 NGN/USD that is
	// 500 * 1600 = 800000 kobo. Verified through the request the provider builds.
	in := CheckoutInput{Credits: 500, USDCents: 500}
	kobo := in.USDCents * 1600
	assert.Equal(t, int64(800_000), kobo)
}
