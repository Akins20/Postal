-- name: CreatePost :one
INSERT INTO posts (workspace_id, author_user_id, status)
VALUES ($1, $2, $3)
RETURNING id, workspace_id, author_user_id, status, created_at, updated_at;

-- name: GetPost :one
SELECT id, workspace_id, author_user_id, status, created_at, updated_at
FROM posts
WHERE id = $1;

-- name: ListPostsByWorkspace :many
SELECT id, workspace_id, author_user_id, status, created_at, updated_at
FROM posts
WHERE workspace_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: UpdatePostStatus :exec
UPDATE posts
SET status = $2, updated_at = now()
WHERE id = $1;

-- name: TouchPost :exec
UPDATE posts
SET updated_at = now()
WHERE id = $1;

-- name: DeletePost :exec
DELETE FROM posts
WHERE id = $1;

-- name: CreatePostVariant :one
INSERT INTO post_variants (post_id, channel_id, body, media_refs, platform_options)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, post_id, channel_id, body, media_refs, platform_options, created_at, updated_at;

-- name: ListVariantsByPost :many
SELECT id, post_id, channel_id, body, media_refs, platform_options, created_at, updated_at
FROM post_variants
WHERE post_id = $1
ORDER BY created_at;

-- name: GetPostVariant :one
SELECT id, post_id, channel_id, body, media_refs, platform_options, created_at, updated_at
FROM post_variants
WHERE id = $1;

-- name: GetVariantByPostChannel :one
SELECT id, post_id, channel_id, body, media_refs, platform_options, created_at, updated_at
FROM post_variants
WHERE post_id = $1 AND channel_id = $2;

-- name: UpdatePostVariant :one
UPDATE post_variants
SET body = $2, media_refs = $3, platform_options = $4, updated_at = now()
WHERE id = $1
RETURNING id, post_id, channel_id, body, media_refs, platform_options, created_at, updated_at;

-- name: DeletePostVariant :exec
DELETE FROM post_variants
WHERE id = $1;

-- name: DeleteVariantsForPost :exec
DELETE FROM post_variants
WHERE post_id = $1;
