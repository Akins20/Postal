-- name: CreateUser :one
INSERT INTO users (email, password_hash)
VALUES ($1, $2)
RETURNING id, email, password_hash, email_verified, status, created_at, updated_at;

-- name: GetUserByEmail :one
SELECT id, email, password_hash, email_verified, status, created_at, updated_at
FROM users
WHERE email = $1;

-- name: GetUserByID :one
SELECT id, email, password_hash, email_verified, status, created_at, updated_at
FROM users
WHERE id = $1;

-- name: SetEmailVerified :exec
UPDATE users
SET email_verified = TRUE, updated_at = now()
WHERE id = $1;

-- name: UpdatePasswordHash :exec
UPDATE users
SET password_hash = $2, updated_at = now()
WHERE id = $1;
