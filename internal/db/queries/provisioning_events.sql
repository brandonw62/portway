-- name: CreateProvisioningEvent :one
INSERT INTO provisioning_events (resource_id, type, old_status, new_status, message, detail, actor_id)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: ListProvisioningEvents :many
SELECT * FROM provisioning_events
WHERE resource_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;
