// Package billing implements the workspace wallet (Phase 13): X/Twitter is the
// only pay-per-use platform, so workspaces pre-fund credits via Stripe or
// Paystack and each successful X publish deducts a configured cost. All other
// platforms are free and bypass billing entirely. The ledger is append-only
// and unique per (workspace, kind, reference) so webhook retries and job
// re-claims stay idempotent.
package billing

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

// ErrInsufficientCredits is returned when a charge or affordability check
// fails because the wallet balance is below the required cost.
var ErrInsufficientCredits = errors.New("insufficient wallet credits")

// Ledger entry kinds.
const (
	KindTopup         = "topup"
	KindPublishCharge = "publish_charge"
	KindRefund        = "refund"
	KindAdjustment    = "adjustment"
)

// Wallet is a workspace's credit balance.
type Wallet struct {
	WorkspaceID  uuid.UUID        `json:"workspace_id"`
	Balance      int64            `json:"balance"`
	PublishCosts map[string]int64 `json:"publish_costs"`
	UpdatedAt    time.Time        `json:"updated_at"`
}

// LedgerEntry is one append-only wallet movement.
type LedgerEntry struct {
	ID          uuid.UUID `json:"id"`
	WorkspaceID uuid.UUID `json:"workspace_id"`
	Kind        string    `json:"kind"`
	Credits     int64     `json:"credits"`
	Reference   string    `json:"reference"`
	Note        string    `json:"note"`
	CreatedAt   time.Time `json:"created_at"`
}

// Pricing holds the credit economics (from config; see docs/BILLING_PLAN.md).
// Costs are tiered by content because X bills URL posts (~$0.20) far above
// plain or media posts (~$0.015 + upload requests): base < media < URL, and
// the highest applicable tier wins.
type Pricing struct {
	// CreditsPerUSDCent converts a top-up amount to credits (default 1).
	CreditsPerUSDCent int64
	// PublishCosts maps a platform key to its base per-publish cost in
	// credits. Platforms absent from every cost map are free.
	PublishCosts map[string]int64
	// MediaCosts is the per-publish cost when the post carries media.
	MediaCosts map[string]int64
	// URLCosts is the per-publish cost when the post body contains a link.
	URLCosts map[string]int64
	// MinTopupCredits is the smallest accepted top-up.
	MinTopupCredits int64
	// NGNPerUSD converts USD pricing to NGN for Paystack charges.
	NGNPerUSD int64
	// ReturnURL is where checkout sends the browser back (the wallet page).
	ReturnURL string
}

// PublishItem describes one variant about to publish, for pricing.
type PublishItem struct {
	Platform string
	Body     string
	HasMedia bool
}

// BodyHasURL reports whether post text contains a web link (X bills these at
// the URL rate).
func BodyHasURL(body string) bool {
	return strings.Contains(body, "http://") || strings.Contains(body, "https://")
}

// CostForItem returns the per-publish cost for one variant: the highest
// applicable tier (URL > media > base). 0 = free platform.
func (p Pricing) CostForItem(it PublishItem) int64 {
	cost := p.PublishCosts[it.Platform]
	if it.HasMedia {
		if c := p.MediaCosts[it.Platform]; c > cost {
			cost = c
		}
	}
	if BodyHasURL(it.Body) {
		if c := p.URLCosts[it.Platform]; c > cost {
			cost = c
		}
	}
	return cost
}

// USDCents converts credits to the USD amount (in cents) a buyer pays.
func (p Pricing) USDCents(credits int64) int64 {
	per := p.CreditsPerUSDCent
	if per <= 0 {
		per = 1
	}
	cents := credits / per
	if credits%per != 0 {
		cents++
	}
	return cents
}
