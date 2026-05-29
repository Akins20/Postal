-- Phase 2 identity & tenancy schema: users, workspaces, capability-based
-- memberships, and single-use email-verification / password-reset tokens.
-- Refresh tokens are NOT stored here; they live in Redis (see internal/auth).
-- Token tables store only a hash of the token, never the plaintext.

-- +goose Up
-- +goose StatementBegin
CREATE TABLE users (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email          TEXT NOT NULL UNIQUE,
    password_hash  TEXT NOT NULL,
    email_verified BOOLEAN NOT NULL DEFAULT FALSE,
    status         TEXT NOT NULL DEFAULT 'active',
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TABLE workspaces (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name          TEXT NOT NULL,
    owner_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    plan          TEXT NOT NULL DEFAULT 'free',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX idx_workspaces_owner ON workspaces (owner_user_id);
-- +goose StatementEnd

-- +goose StatementBegin
-- role is the named preset; permissions is the authoritative capability set
-- (may diverge from the preset for fine-grained per-user grants). See
-- docs/MASTER_PLAN.md 5.1.
CREATE TABLE workspace_members (
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    user_id      UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role         TEXT NOT NULL,
    permissions  TEXT[] NOT NULL DEFAULT '{}',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (workspace_id, user_id)
);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX idx_workspace_members_user ON workspace_members (user_id);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TABLE email_verification_tokens (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash TEXT NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    consumed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX idx_email_verification_user ON email_verification_tokens (user_id);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TABLE password_reset_tokens (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash TEXT NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    consumed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX idx_password_reset_user ON password_reset_tokens (user_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE password_reset_tokens;
-- +goose StatementEnd
-- +goose StatementBegin
DROP TABLE email_verification_tokens;
-- +goose StatementEnd
-- +goose StatementBegin
DROP TABLE workspace_members;
-- +goose StatementEnd
-- +goose StatementBegin
DROP TABLE workspaces;
-- +goose StatementEnd
-- +goose StatementBegin
DROP TABLE users;
-- +goose StatementEnd
