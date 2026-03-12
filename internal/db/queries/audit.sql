-- name: CreateAuditEntry :one
INSERT INTO audit_entries (actor_id, project_id, action, target_type, target_id, detail, allowed)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: ListAuditEntries :many
SELECT * FROM audit_entries
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: ListAuditEntriesByProject :many
SELECT * FROM audit_entries
WHERE project_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: ListAuditEntriesByActor :many
SELECT * FROM audit_entries
WHERE actor_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;
