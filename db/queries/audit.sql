-- name: InsertAuditLog :one
INSERT INTO audit_log (workspace_id, actor_user_id, action, target, metadata, ip)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, created_at;

-- name: ListAuditLogByWorkspace :many
SELECT id, workspace_id, actor_user_id, action, target, metadata, ip, created_at
FROM audit_log
WHERE workspace_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: ListWorkspaceActivity :many
SELECT a.id, a.actor_user_id, u.email AS actor_email, a.action, a.target,
       a.metadata, a.created_at
FROM audit_log a
LEFT JOIN users u ON u.id = a.actor_user_id
WHERE a.workspace_id = $1
ORDER BY a.id DESC
LIMIT $2;
