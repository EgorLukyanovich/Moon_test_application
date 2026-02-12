-- name: CreateTeam :execresult
INSERT INTO teams (name, created_by) 
VALUES (?, ?);

-- name: GetTeamByID :one
SELECT * FROM teams 
WHERE id = ? LIMIT 1;

-- name: AddTeamMember :exec
INSERT INTO team_members (team_id, user_id, role) 
VALUES (?, ?, ?);

-- name: GetUserRoleInTeam :one
SELECT role FROM team_members 
WHERE team_id = ? AND user_id = ?;

-- name: ListUserTeams :many
SELECT t.id, t.name, t.created_by, t.created_at, tm.role 
FROM teams t
JOIN team_members tm ON t.id = tm.team_id
WHERE tm.user_id = ?;