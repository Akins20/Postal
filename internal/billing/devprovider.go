package billing

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

// DevProvider is a development-only "payment" provider: it credits the wallet
// immediately and sends the browser straight back to the wallet page. It must
// NEVER be registered outside POSTAL_ENV=development (serve wiring enforces
// this) — it is free money by design, for local testing of the full flow.
type DevProvider struct {
	credit func(ctx context.Context, workspaceID uuid.UUID, kind string, credits int64, reference, note string) (bool, error)
}

// NewDevProvider builds the dev provider over the service's Credit func.
func NewDevProvider(credit func(ctx context.Context, workspaceID uuid.UUID, kind string, credits int64, reference, note string) (bool, error)) *DevProvider {
	return &DevProvider{credit: credit}
}

// Name implements Provider.
func (p *DevProvider) Name() string { return "dev" }

// CreateCheckout credits instantly (idempotent by a random reference) and
// returns the success URL — no hosted page involved.
func (p *DevProvider) CreateCheckout(ctx context.Context, in CheckoutInput) (string, error) {
	ref := "dev:" + uuid.NewString()
	if _, err := p.credit(ctx, in.WorkspaceID, KindTopup, in.Credits, ref, "dev top-up (no real payment)"); err != nil {
		return "", fmt.Errorf("dev credit: %w", err)
	}
	return in.ReturnURL + "?status=success&provider=dev", nil
}
