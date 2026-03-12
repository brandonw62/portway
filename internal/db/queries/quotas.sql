-- name: GetQuota :one
SELECT * FROM quotas WHERE id = $1;

-- name: ListQuotasForProject :many
SELECT * FROM quotas
WHERE project_id = $1 OR project_id IS NULL
ORDER BY project_id NULLS LAST, resource_type;

-- name: CreateQuota :one
INSERT INTO quotas (project_id, resource_type, "limit")
VALUES ($1, $2, $3)
RETURNING *;

-- name: UpdateQuota :one
UPDATE quotas
SET "limit" = $2, updated_at = now()
WHERE id = $1
RETURNING *;

-- name: DeleteQuota :exec
DELETE FROM quotas WHERE id = $1;
