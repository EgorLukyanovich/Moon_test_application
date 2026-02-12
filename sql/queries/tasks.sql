-- name: CreateTask :execresult
INSERT INTO tasks (title, description, status, team_id, assignee_id, created_by) 
VALUES (?, ?, ?, ?, ?, ?);

-- name: GetTaskByID :one
SELECT * FROM tasks 
WHERE id = ? LIMIT 1;

-- name: UpdateTask :exec
UPDATE tasks 
SET title = ?, description = ?, status = ?, assignee_id = ? 
WHERE id = ?;

-- name: DeleteTask :exec
DELETE FROM tasks WHERE id = ?;

-- name: ListTasks :many
SELECT * FROM tasks
WHERE 
    team_id = ?
    AND (sqlc.narg('status') IS NULL OR status = sqlc.narg('status'))
    AND (sqlc.narg('assignee_id') IS NULL OR assignee_id = sqlc.narg('assignee_id'))
ORDER BY created_at DESC
LIMIT ? OFFSET ?;