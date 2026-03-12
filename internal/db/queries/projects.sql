-- name: GetProject :one
SELECT * FROM projects WHERE id = $1;

-- name: GetProjectBySlug :one
SELECT * FROM projects WHERE team_id = $1 AND slug = $2;

-- name: ListProjectsByTeam :many
SELECT * FROM projects WHERE team_id = $1 ORDER BY name LIMIT $2 OFFSET $3;

-- name: CreateProject :one
INSERT INTO projects (team_id, name, slug, description)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: UpdateProject :one
UPDATE projects
SET name = $2, slug = $3, description = $4, updated_at = now()
WHERE id = $1
RETURNING *;

-- name: DeleteProject :exec
DELETE FROM projects WHERE id = $1;

-- name: AddMembership :one
INSERT INTO memberships (user_id, project_id, role)
VALUES ($1, $2, $3)
ON CONFLICT (user_id, project_id) DO UPDATE SET role = EXCLUDED.role, updated_at = now()
RETURNING *;

-- name: RemoveMembership :exec
DELETE FROM memberships WHERE user_id = $1 AND project_id = $2;

-- name: GetMembership :one
SELECT * FROM memberships WHERE user_id = $1 AND project_id = $2;

-- name: ListProjectMembers :many
SELECT m.*, u.email, u.name AS user_name
FROM memberships m
JOIN users u ON u.id = m.user_id
WHERE m.project_id = $1
ORDER BY m.created_at;
