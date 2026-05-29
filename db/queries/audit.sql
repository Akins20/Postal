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
