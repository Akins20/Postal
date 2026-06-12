-- name: UpsertWallet :one
-- Ensure the workspace has a wallet row and return it.
INSERT INTO wallets (workspace_id)
VALUES ($1)
ON CONFLICT (workspace_id) DO UPDATE SET workspace_id = EXCLUDED.workspace_id
RETURNING workspace_id, balance, updated_at;

-- name: GetWalletBalance :one
SELECT COALESCE(
    (SELECT balance FROM wallets WHERE workspace_id = $1),
    0
)::BIGINT AS balance;

-- name: CreditWallet :one
-- Add credits (topup/refund). The wallet row is created if missing.
INSERT INTO wallets (workspace_id, balance, updated_at)
VALUES ($1, $2, now())
ON CONFLICT (workspace_id)
DO UPDATE SET balance = wallets.balance + EXCLUDED.balance, updated_at = now()
RETURNING workspace_id, balance, updated_at;

-- name: DebitWalletIfEnough :one
-- Atomically deduct; returns no row when funds are insufficient.
UPDATE wallets
SET balance = balance - $2, updated_at = now()
WHERE workspace_id = $1 AND balance >= $2
RETURNING workspace_id, balance, updated_at;

-- name: InsertLedgerEntry :one
-- Append a ledger entry. ON CONFLICT DO NOTHING + no row returned signals a
-- duplicate (webhook retry / double claim) so the caller can skip the credit.
INSERT INTO wallet_ledger (workspace_id, kind, credits, reference, note)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (workspace_id, kind, reference) DO NOTHING
RETURNING id, workspace_id, kind, credits, reference, note, created_at;

-- name: ListLedgerEntries :many
SELECT id, workspace_id, kind, credits, reference, note, created_at
FROM wallet_ledger
WHERE workspace_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;
