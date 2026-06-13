-- name: CreateWorkspace :one
INSERT INTO workspaces (name, owner_user_id)
VALUES ($1, $2)
RETURNING id, name, owner_user_id, plan, created_at;

-- name: ListWorkspacesForUser :many
SELECT w.id, w.name, w.owner_user_id, w.plan, w.created_at
FROM workspaces w
JOIN workspace_members m ON m.workspace_id = w.id
WHERE m.user_id = $1
ORDER BY w.created_at;

-- name: CreateMember :one
INSERT INTO workspace_members (workspace_id, user_id, role, permissions)
VALUES ($1, $2, $3, $4)
RETURNING workspace_id, user_id, role, permissions, created_at, channel_restricted;

-- name: GetMember :one
SELECT workspace_id, user_id, role, permissions, created_at, channel_restricted
FROM workspace_members
WHERE workspace_id = $1 AND user_id = $2;

-- name: UpdateMemberPermissions :one
UPDATE workspace_members
SET role = $3, permissions = $4
WHERE workspace_id = $1 AND user_id = $2
RETURNING workspace_id, user_id, role, permissions, created_at, channel_restricted;

-- name: ListMembers :many
SELECT workspace_id, user_id, role, permissions, created_at, channel_restricted
FROM workspace_members
WHERE workspace_id = $1
ORDER BY created_at;
