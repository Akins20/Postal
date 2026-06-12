-- Phase 13: wallet billing (X-exclusive pay-per-use). Workspaces pre-fund a
-- credits wallet; successful X publishes deduct, payment-provider webhooks
-- credit. The ledger is append-only and (workspace, kind, reference)-unique
-- so webhook retries and job re-claims stay idempotent.

-- +goose Up
-- +goose StatementBegin
CREATE TABLE wallets (
    workspace_id  UUID PRIMARY KEY REFERENCES workspaces(id) ON DELETE CASCADE,
    balance       BIGINT NOT NULL DEFAULT 0 CHECK (balance >= 0),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TABLE wallet_ledger (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id  UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    kind          TEXT NOT NULL,   -- topup | publish_charge | refund | adjustment
    credits       BIGINT NOT NULL, -- signed: topup/refund > 0, publish_charge < 0
    reference     TEXT NOT NULL,   -- provider event id / scheduled job id
    note          TEXT NOT NULL DEFAULT '',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (workspace_id, kind, reference)
);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX idx_wallet_ledger_workspace ON wallet_ledger (workspace_id, created_at DESC);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE wallet_ledger;
-- +goose StatementEnd
-- +goose StatementBegin
DROP TABLE wallets;
-- +goose StatementEnd
