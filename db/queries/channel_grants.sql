-- name: SetMemberChannelRestricted :exec
UPDATE workspace_members SET channel_restricted = $3
WHERE workspace_id = $1 AND user_id = $2;

-- name: GetMemberChannelRestricted :one
SELECT channel_restricted FROM workspace_members
WHERE workspace_id = $1 AND user_id = $2;

-- name: DeleteChannelGrantsForUser :exec
DELETE FROM channel_grants WHERE workspace_id = $1 AND user_id = $2;

-- name: InsertChannelGrant :exec
INSERT INTO channel_grants (workspace_id, channel_id, user_id)
VALUES ($1, $2, $3)
ON CONFLICT (channel_id, user_id) DO NOTHING;

-- name: ListChannelGrantsForUser :many
SELECT channel_id FROM channel_grants
WHERE workspace_id = $1 AND user_id = $2;

-- name: IsChannelPublishAllowed :one
SELECT (NOT m.channel_restricted)
       OR EXISTS (
         SELECT 1 FROM channel_grants g
         WHERE g.channel_id = sqlc.arg(channel_id) AND g.user_id = m.user_id
       ) AS allowed
FROM workspace_members m
WHERE m.workspace_id = sqlc.arg(workspace_id) AND m.user_id = sqlc.arg(user_id);
