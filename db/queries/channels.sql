-- name: CreateChannel :one
INSERT INTO channels (workspace_id, platform, platform_account_id, handle, display_name, connected_by)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, workspace_id, platform, platform_account_id, handle, display_name, status, connected_by, created_at, updated_at;

-- name: GetChannel :one
SELECT id, workspace_id, platform, platform_account_id, handle, display_name, status, connected_by, created_at, updated_at
FROM channels
WHERE id = $1;

-- name: GetChannelByAccount :one
SELECT id, workspace_id, platform, platform_account_id, handle, display_name, status, connected_by, created_at, updated_at
FROM channels
WHERE workspace_id = $1 AND platform = $2 AND platform_account_id = $3;

-- name: ListChannels :many
SELECT id, workspace_id, platform, platform_account_id, handle, display_name, status, connected_by, created_at, updated_at
FROM channels
WHERE workspace_id = $1
ORDER BY created_at;

-- name: UpdateChannelStatus :exec
UPDATE channels
SET status = $2, updated_at = now()
WHERE id = $1;

-- name: UpdateChannelIdentity :exec
UPDATE channels
SET handle = $2, display_name = $3, updated_at = now()
WHERE id = $1;

-- name: DeleteChannel :exec
DELETE FROM channels
WHERE id = $1;

-- name: UpsertChannelCredential :exec
INSERT INTO channel_credentials (channel_id, encrypted_access_token, encrypted_refresh_token, scopes, expires_at, key_version)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (channel_id) DO UPDATE
SET encrypted_access_token = EXCLUDED.encrypted_access_token,
    encrypted_refresh_token = EXCLUDED.encrypted_refresh_token,
    scopes = EXCLUDED.scopes,
    expires_at = EXCLUDED.expires_at,
    key_version = EXCLUDED.key_version,
    updated_at = now();

-- name: GetChannelCredential :one
SELECT channel_id, encrypted_access_token, encrypted_refresh_token, scopes, expires_at, key_version, created_at, updated_at
FROM channel_credentials
WHERE channel_id = $1;

-- name: DeleteChannelCredential :exec
DELETE FROM channel_credentials
WHERE channel_id = $1;

-- name: ListChannelsDueForRefresh :many
SELECT c.id, c.workspace_id, c.platform, cc.expires_at
FROM channels c
JOIN channel_credentials cc ON cc.channel_id = c.id
WHERE c.status = 'active'
  AND cc.encrypted_refresh_token IS NOT NULL
  AND cc.expires_at IS NOT NULL
  AND cc.expires_at < $1
ORDER BY cc.expires_at
LIMIT $2;

-- name: CountActiveChannelsForWorkspace :one
SELECT count(*) FROM channels WHERE workspace_id = $1 AND status = 'active';
