-- name: GetPolicy :one
SELECT * FROM policies WHERE id = $1;

-- name: ListPolicies :many
SELECT * FROM policies ORDER BY created_at DESC LIMIT $1 OFFSET $2;

-- name: ListActivePoliciesForProject :many
SELECT * FROM policies
WHERE enabled = true AND (scope = 'global' OR project_id = $1)
ORDER BY scope, created_at;

-- name: CreatePolicy :one
INSERT INTO policies (name, description, scope, project_id, enabled, created_by)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: UpdatePolicy :one
UPDATE policies
SET name = $2, description = $3, enabled = $4, updated_at = now()
WHERE id = $1
RETURNING *;

-- name: DeletePolicy :exec
DELETE FROM policies WHERE id = $1;

-- name: ListPolicyRules :many
SELECT * FROM policy_rules WHERE policy_id = $1 ORDER BY resource_type, attribute;

-- name: CreatePolicyRule :one
INSERT INTO policy_rules (policy_id, description, resource_type, attribute, operator, value, effect, metadata)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: DeletePolicyRule :exec
DELETE FROM policy_rules WHERE id = $1;

-- name: DeletePolicyRulesByPolicy :exec
DELETE FROM policy_rules WHERE policy_id = $1;
