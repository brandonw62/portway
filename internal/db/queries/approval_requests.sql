-- name: GetApprovalRequest :one
SELECT * FROM approval_requests WHERE id = $1;

-- name: ListPendingApprovalsByProject :many
SELECT * FROM approval_requests
WHERE project_id = $1 AND status = 'pending'
ORDER BY created_at DESC;

-- name: ListApprovalRequestsByProject :many
SELECT * FROM approval_requests
WHERE project_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: CreateApprovalRequest :one
INSERT INTO approval_requests (
    project_id, requested_by, resource_type, request_payload,
    reasons, matched_policies, status, expires_at
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: ReviewApprovalRequest :one
UPDATE approval_requests
SET status = $2, reviewed_by = $3, review_comment = $4, reviewed_at = now(), updated_at = now()
WHERE id = $1
RETURNING *;

-- name: ListPendingApprovals :many
SELECT * FROM approval_requests
WHERE status = 'pending'
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: ExpireApprovalRequests :execrows
UPDATE approval_requests
SET status = 'expired', updated_at = now()
WHERE status = 'pending' AND expires_at < now();
