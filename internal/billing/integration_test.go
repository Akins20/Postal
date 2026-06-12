package billing_test

import (
	"context"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Akins20/postal/internal/billing"
	"github.com/Akins20/postal/internal/platform/db"
	"github.com/Akins20/postal/internal/platform/db/sqlc"
)

// setup connects Postgres from env and seeds a workspace + an X channel. It
// skips when the database is unavailable so unit runs stay green.
func setup(t *testing.T) (*billing.Service, uuid.UUID, uuid.UUID) {
	t.Helper()
	dsn := os.Getenv("POSTAL_DATABASE_URL")
	if dsn == "" {
		t.Skip("POSTAL_DATABASE_URL not set; skipping billing integration test")
	}
	ctx := context.Background()
	pool, err := db.Connect(ctx, dsn)
	if err != nil {
		t.Skipf("postgres unreachable: %v", err)
	}
	t.Cleanup(pool.Close)

	q := pool.Queries()
	user, err := q.CreateUser(ctx, sqlc.CreateUserParams{Email: "billing-" + uuid.NewString() + "@example.com", PasswordHash: "x"})
	require.NoError(t, err)
	ws, err := q.CreateWorkspace(ctx, sqlc.CreateWorkspaceParams{Name: "Billing", OwnerUserID: user.ID})
	require.NoError(t, err)
	ch, err := q.CreateChannel(ctx, sqlc.CreateChannelParams{
		WorkspaceID: ws.ID, Platform: "twitter", PlatformAccountID: "acct-" + uuid.NewString(),
		Handle: "@bill", DisplayName: "Bill", ConnectedBy: &user.ID,
	})
	require.NoError(t, err)

	svc := billing.NewService(pool, billing.Pricing{
		CreditsPerUSDCent: 1,
		PublishCosts:      map[string]int64{"twitter": 25},
		MinTopupCredits:   100,
	}, nil, nil)
	return svc, ws.ID, ch.ID
}

func TestCreditIsIdempotentByReference(t *testing.T) {
	svc, wsID, _ := setup(t)
	ctx := context.Background()

	applied, err := svc.Credit(ctx, wsID, billing.KindTopup, 500, "stripe:evt_dup", "topup")
	require.NoError(t, err)
	assert.True(t, applied, "first credit applies")

	applied, err = svc.Credit(ctx, wsID, billing.KindTopup, 500, "stripe:evt_dup", "topup")
	require.NoError(t, err)
	assert.False(t, applied, "replayed webhook must not double-credit")

	w, err := svc.Wallet(ctx, wsID)
	require.NoError(t, err)
	assert.Equal(t, int64(500), w.Balance)
}

func TestChargeRefundLifecycle(t *testing.T) {
	svc, wsID, chID := setup(t)
	ctx := context.Background()
	jobID := uuid.New()

	// No funds: schedule gate and charge both refuse.
	err := svc.CheckAffordable(ctx, wsID, []string{"twitter"})
	assert.ErrorIs(t, err, billing.ErrInsufficientCredits)
	err = svc.ChargePublish(ctx, jobID, chID)
	assert.ErrorIs(t, err, billing.ErrInsufficientCredits)

	// Fund it; the same charge now succeeds, and re-claiming never double-charges.
	_, err = svc.Credit(ctx, wsID, billing.KindTopup, 100, "stripe:evt_fund", "topup")
	require.NoError(t, err)
	require.NoError(t, svc.CheckAffordable(ctx, wsID, []string{"twitter"}))
	require.NoError(t, svc.ChargePublish(ctx, jobID, chID))
	require.NoError(t, svc.ChargePublish(ctx, jobID, chID), "idempotent re-charge")

	w, err := svc.Wallet(ctx, wsID)
	require.NoError(t, err)
	assert.Equal(t, int64(75), w.Balance, "exactly one 25-credit charge")

	// Terminal failure refunds once.
	require.NoError(t, svc.RefundPublish(ctx, jobID, chID))
	require.NoError(t, svc.RefundPublish(ctx, jobID, chID), "idempotent refund")
	w, err = svc.Wallet(ctx, wsID)
	require.NoError(t, err)
	assert.Equal(t, int64(100), w.Balance)

	// Free platforms bypass billing entirely.
	require.NoError(t, svc.CheckAffordable(ctx, wsID, []string{"mastodon"}))
}

func TestLedgerListsMovements(t *testing.T) {
	svc, wsID, _ := setup(t)
	ctx := context.Background()
	_, err := svc.Credit(ctx, wsID, billing.KindTopup, 300, "paystack:ref_1", "topup")
	require.NoError(t, err)

	entries, err := svc.Ledger(ctx, wsID, 10, 0)
	require.NoError(t, err)
	require.NotEmpty(t, entries)
	assert.Equal(t, billing.KindTopup, entries[0].Kind)
	assert.Equal(t, int64(300), entries[0].Credits)
}
