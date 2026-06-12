package billing

import (
	"context"
	"errors"

	"github.com/google/uuid"
)

// ErrProviderUnavailable is returned when a checkout names a provider that
// isn't configured (missing keys, or "dev" outside development).
var ErrProviderUnavailable = errors.New("payment provider unavailable")

// ErrBadTopup is returned for invalid top-up requests (below minimum, etc.).
var ErrBadTopup = errors.New("invalid top-up")

// CheckoutInput is what a provider needs to start a hosted checkout. USDCents
// is derived server-side from Credits — never from client-supplied amounts.
type CheckoutInput struct {
	WorkspaceID uuid.UUID
	Credits     int64
	USDCents    int64
	ReturnURL   string
}

// Provider starts a hosted checkout and returns the redirect URL. Crediting
// happens later via the provider's signed webhook, never on redirect.
type Provider interface {
	// Name is the provider key used in topup requests ("stripe", "paystack", "dev").
	Name() string
	// CreateCheckout returns the URL to send the buyer's browser to.
	CreateCheckout(ctx context.Context, in CheckoutInput) (string, error)
}
