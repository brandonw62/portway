-- name: GetResource :one
SELECT * FROM resources WHERE id = $1;

-- name: GetResourceBySlug :one
SELECT * FROM resources WHERE project_id = $1 AND slug = $2;

-- name: ListResourcesByProject :many
SELECT * FROM resources WHERE project_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3;

-- name: CountResourcesByProjectAndType :one
SELECT count(*) FROM resources
WHERE project_id = $1 AND resource_type_id = $2 AND status NOT IN ('deleted', 'failed');

-- name: ListResourcesByStatus :many
SELECT * FROM resources
WHERE status = $1
ORDER BY updated_at ASC
LIMIT $2 OFFSET $3;

-- name: CreateResource :one
INSERT INTO resources (project_id, resource_type_id, name, slug, status, spec, requested_by)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: UpdateResourceStatus :one
UPDATE resources
SET status = $2, status_message = $3, updated_at = now()
WHERE id = $1
RETURNING *;

-- name: UpdateResourceSpec :one
UPDATE resources
SET spec = $2, updated_at = now()
WHERE id = $1
RETURNING *;

-- name: SetResourceProviderRef :exec
UPDATE resources SET provider_ref = $2, updated_at = now() WHERE id = $1;

-- name: DeleteResource :exec
DELETE FROM resources WHERE id = $1;
