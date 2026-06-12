package billing

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/Akins20/postal/internal/platform/db"
	"github.com/Akins20/postal/internal/platform/db/sqlc"
)

// Service owns wallet state: reads, credits from payment webhooks, atomic
// publish charges, and refunds. All movements append a ledger entry whose
// (workspace, kind, reference) uniqueness makes retries idempotent.
type Service struct {
	pool      *db.Pool
	pricing   Pricing
	providers map[string]Provider
	log       *slog.Logger
}

// NewService builds the billing service. providers maps a provider key
// ("stripe", "paystack", "dev") to its checkout implementation.
func NewService(pool *db.Pool, pricing Pricing, providers map[string]Provider, log *slog.Logger) *Service {
	return &Service{pool: pool, pricing: pricing, providers: providers, log: log}
}

// Pricing exposes the configured credit economics (for handlers/UI).
func (s *Service) Pricing() Pricing { return s.pricing }

// Wallet returns the workspace's balance plus the publish price list.
func (s *Service) Wallet(ctx context.Context, workspaceID uuid.UUID) (*Wallet, error) {
	w, err := s.pool.Queries().UpsertWallet(ctx, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("loading wallet: %w", err)
	}
	costs := map[string]int64{}
	for k, v := range s.pricing.PublishCosts {
		costs[k] = v
	}
	for k, v := range s.pricing.MediaCosts {
		costs[k+"_media"] = v
	}
	for k, v := range s.pricing.URLCosts {
		costs[k+"_url"] = v
	}
	return &Wallet{
		WorkspaceID:  w.WorkspaceID,
		Balance:      w.Balance,
		PublishCosts: costs,
		UpdatedAt:    w.UpdatedAt.Time,
	}, nil
}

// Ledger lists the workspace's wallet movements, newest first.
func (s *Service) Ledger(ctx context.Context, workspaceID uuid.UUID, limit, offset int32) ([]LedgerEntry, error) {
	rows, err := s.pool.Queries().ListLedgerEntries(ctx, sqlc.ListLedgerEntriesParams{
		WorkspaceID: workspaceID, Limit: limit, Offset: offset,
	})
	if err != nil {
		return nil, fmt.Errorf("listing ledger: %w", err)
	}
	out := make([]LedgerEntry, 0, len(rows))
	for _, r := range rows {
		out = append(out, LedgerEntry{
			ID: r.ID, WorkspaceID: r.WorkspaceID, Kind: r.Kind,
			Credits: r.Credits, Reference: r.Reference, Note: r.Note, CreatedAt: r.CreatedAt.Time,
		})
	}
	return out, nil
}

// CreateCheckout starts a top-up with the chosen provider and returns the URL
// to send the browser to. The charge amount is derived server-side from the
// requested credits — client-supplied money amounts are never trusted.
func (s *Service) CreateCheckout(ctx context.Context, workspaceID uuid.UUID, provider string, credits int64) (string, error) {
	p, ok := s.providers[provider]
	if !ok {
		return "", fmt.Errorf("payment provider %q is not configured: %w", provider, ErrProviderUnavailable)
	}
	if credits < s.pricing.MinTopupCredits {
		return "", fmt.Errorf("minimum top-up is %d credits: %w", s.pricing.MinTopupCredits, ErrBadTopup)
	}
	url, err := p.CreateCheckout(ctx, CheckoutInput{
		WorkspaceID: workspaceID,
		Credits:     credits,
		USDCents:    s.pricing.USDCents(credits),
		ReturnURL:   s.pricing.ReturnURL,
	})
	if err != nil {
		return "", fmt.Errorf("creating %s checkout: %w", provider, err)
	}
	return url, nil
}

// Credit applies a positive wallet movement (top-up/adjustment), idempotent by
// (kind, reference): a replayed webhook credits nothing and returns false.
func (s *Service) Credit(ctx context.Context, workspaceID uuid.UUID, kind string, credits int64, reference, note string) (applied bool, err error) {
	if credits <= 0 {
		return false, fmt.Errorf("credit amount must be positive: %w", ErrBadTopup)
	}
	err = s.pool.WithTx(ctx, func(q *sqlc.Queries) error {
		_, lErr := q.InsertLedgerEntry(ctx, sqlc.InsertLedgerEntryParams{
			WorkspaceID: workspaceID, Kind: kind, Credits: credits, Reference: reference, Note: note,
		})
		if errors.Is(lErr, pgx.ErrNoRows) {
			return nil // duplicate reference -> already applied
		}
		if lErr != nil {
			return fmt.Errorf("appending ledger: %w", lErr)
		}
		applied = true
		if _, cErr := q.CreditWallet(ctx, sqlc.CreditWalletParams{WorkspaceID: workspaceID, Balance: credits}); cErr != nil {
			return fmt.Errorf("crediting wallet: %w", cErr)
		}
		return nil
	})
	if err != nil {
		return false, err
	}
	if s.log != nil && applied {
		s.log.InfoContext(ctx, "wallet credited",
			slog.String("workspace_id", workspaceID.String()),
			slog.Int64("credits", credits), slog.String("kind", kind), slog.String("reference", reference))
	}
	return applied, nil
}

// CheckAffordable verifies the workspace can cover publishing the given
// variants (the schedule-time soft gate). Free platforms cost nothing; URL
// and media posts use their tier prices.
func (s *Service) CheckAffordable(ctx context.Context, workspaceID uuid.UUID, items []PublishItem) error {
	var total int64
	for _, it := range items {
		total += s.pricing.CostForItem(it)
	}
	if total == 0 {
		return nil
	}
	balance, err := s.pool.Queries().GetWalletBalance(ctx, workspaceID)
	if err != nil {
		return fmt.Errorf("reading balance: %w", err)
	}
	if balance < total {
		return fmt.Errorf("need %d credits, have %d: %w", total, balance, ErrInsufficientCredits)
	}
	return nil
}

// ChargePublish deducts the content-tiered platform cost for one publish job,
// atomically and idempotently (reference = job ID). Free platforms are a
// no-op. It resolves the channel to find the workspace and platform.
func (s *Service) ChargePublish(ctx context.Context, jobID, channelID uuid.UUID, body string, hasMedia bool) error {
	ch, err := s.pool.Queries().GetChannel(ctx, channelID)
	if err != nil {
		return fmt.Errorf("loading channel for billing: %w", err)
	}
	item := PublishItem{Platform: ch.Platform, Body: body, HasMedia: hasMedia}
	cost := s.pricing.CostForItem(item)
	if cost == 0 {
		return nil
	}
	note := "publish to " + ch.Platform
	switch {
	case BodyHasURL(body):
		note += " (link post)"
	case hasMedia:
		note += " (media post)"
	}
	return s.pool.WithTx(ctx, func(q *sqlc.Queries) error {
		_, lErr := q.InsertLedgerEntry(ctx, sqlc.InsertLedgerEntryParams{
			WorkspaceID: ch.WorkspaceID, Kind: KindPublishCharge, Credits: -cost,
			Reference: jobID.String(), Note: note,
		})
		if errors.Is(lErr, pgx.ErrNoRows) {
			return nil // this job was already charged (re-claim after crash)
		}
		if lErr != nil {
			return fmt.Errorf("appending charge ledger: %w", lErr)
		}
		_, dErr := q.DebitWalletIfEnough(ctx, sqlc.DebitWalletIfEnoughParams{
			WorkspaceID: ch.WorkspaceID, Balance: cost,
		})
		if errors.Is(dErr, pgx.ErrNoRows) {
			return ErrInsufficientCredits // rolls back the ledger entry too
		}
		if dErr != nil {
			return fmt.Errorf("debiting wallet: %w", dErr)
		}
		return nil
	})
}

// RefundPublish returns a job's charge after a terminal publish failure,
// idempotent by job ID. The amount comes from the recorded charge ledger
// entry (never recomputed, so tier/config changes can't skew refunds). A job
// that was never charged refunds nothing.
func (s *Service) RefundPublish(ctx context.Context, jobID, channelID uuid.UUID) error {
	ch, err := s.pool.Queries().GetChannel(ctx, channelID)
	if err != nil {
		return fmt.Errorf("loading channel for refund: %w", err)
	}
	entry, err := s.pool.Queries().GetLedgerEntryByRef(ctx, sqlc.GetLedgerEntryByRefParams{
		WorkspaceID: ch.WorkspaceID, Kind: KindPublishCharge, Reference: jobID.String(),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil // never charged (free platform, or failed before the charge)
	}
	if err != nil {
		return fmt.Errorf("loading charge for refund: %w", err)
	}
	_, err = s.Credit(ctx, ch.WorkspaceID, KindRefund, -entry.Credits, jobID.String(), "refund: publish failed")
	return err
}
