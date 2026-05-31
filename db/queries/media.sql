-- name: CreateMediaAsset :one
INSERT INTO media_assets (workspace_id, kind, storage_key, mime, width, height, duration_ms, bytes, status)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING id, workspace_id, kind, storage_key, mime, width, height, duration_ms, bytes, status, created_at;

-- name: GetMediaAsset :one
SELECT id, workspace_id, kind, storage_key, mime, width, height, duration_ms, bytes, status, created_at
FROM media_assets
WHERE id = $1;

-- name: ListMediaAssets :many
SELECT id, workspace_id, kind, storage_key, mime, width, height, duration_ms, bytes, status, created_at
FROM media_assets
WHERE workspace_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: DeleteMediaAsset :exec
DELETE FROM media_assets
WHERE id = $1;

-- name: SumMediaBytesForWorkspace :one
SELECT COALESCE(SUM(bytes), 0)::BIGINT AS total
FROM media_assets
WHERE workspace_id = $1;

-- name: LockWorkspaceForUpdate :exec
-- Row-locks the workspace so a quota check + media insert is serialized against
-- concurrent uploads in the same workspace (prevents TOCTOU quota overshoot).
SELECT id FROM workspaces WHERE id = $1 FOR UPDATE;
