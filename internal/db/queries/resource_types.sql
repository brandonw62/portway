-- name: GetResourceType :one
SELECT * FROM resource_types WHERE id = $1;

-- name: GetResourceTypeBySlug :one
SELECT * FROM resource_types WHERE slug = $1;

-- name: ListResourceTypes :many
SELECT * FROM resource_types WHERE enabled = true ORDER BY category, name;

-- name: ListAllResourceTypes :many
SELECT * FROM resource_types ORDER BY category, name;

-- name: CreateResourceType :one
INSERT INTO resource_types (name, slug, category, description, default_spec, spec_schema, enabled)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: UpdateResourceType :one
UPDATE resource_types
SET name = $2, slug = $3, category = $4, description = $5,
    default_spec = $6, spec_schema = $7, enabled = $8, updated_at = now()
WHERE id = $1
RETURNING *;

-- name: DeleteResourceType :exec
DELETE FROM resource_types WHERE id = $1;
