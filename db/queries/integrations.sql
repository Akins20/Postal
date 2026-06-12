-- name: UpsertWorkspaceIntegration :one
INSERT INTO workspace_integrations (workspace_id, provider, enabled, auto_apply, credentials, updated_at)
VALUES ($1, $2, $3, $4, $5, now())
ON CONFLICT (workspace_id, provider)
DO UPDATE SET enabled = EXCLUDED.enabled, auto_apply = EXCLUDED.auto_apply,
              credentials = COALESCE(EXCLUDED.credentials, workspace_integrations.credentials),
              updated_at = now()
RETURNING workspace_id, provider, enabled, auto_apply, credentials, created_at, updated_at;

-- name: GetWorkspaceIntegration :one
SELECT workspace_id, provider, enabled, auto_apply, credentials, created_at, updated_at
FROM workspace_integrations
WHERE workspace_id = $1 AND provider = $2;

-- name: ListWorkspaceIntegrations :many
SELECT workspace_id, provider, enabled, auto_apply, credentials, created_at, updated_at
FROM workspace_integrations
WHERE workspace_id = $1
ORDER BY provider;
