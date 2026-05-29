-- name: CreateEmailVerificationToken :one
INSERT INTO email_verification_tokens (user_id, token_hash, expires_at)
VALUES ($1, $2, $3)
RETURNING id, user_id, token_hash, expires_at, consumed_at, created_at;

-- name: GetEmailVerificationToken :one
SELECT id, user_id, token_hash, expires_at, consumed_at, created_at
FROM email_verification_tokens
WHERE token_hash = $1;

-- name: ConsumeEmailVerificationToken :exec
UPDATE email_verification_tokens
SET consumed_at = now()
WHERE id = $1;

-- name: CreatePasswordResetToken :one
INSERT INTO password_reset_tokens (user_id, token_hash, expires_at)
VALUES ($1, $2, $3)
RETURNING id, user_id, token_hash, expires_at, consumed_at, created_at;

-- name: GetPasswordResetToken :one
SELECT id, user_id, token_hash, expires_at, consumed_at, created_at
FROM password_reset_tokens
WHERE token_hash = $1;

-- name: ConsumePasswordResetToken :exec
UPDATE password_reset_tokens
SET consumed_at = now()
WHERE id = $1;
