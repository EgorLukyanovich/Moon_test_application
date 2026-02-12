-- name: CreateTaskHistory :exec
INSERT INTO task_history (task_id, changed_by, change_type, old_value, new_value)
VALUES (?, ?, ?, ?, ?);

-- name: ListTaskHistory :many
SELECT th.*, u.email as user_email
FROM task_history th
LEFT JOIN users u ON th.changed_by = u.id
WHERE th.task_id = ?
ORDER BY th.created_at DESC;

-- name: CreateTaskComment :execresult
INSERT INTO task_comments (task_id, user_id, content)
VALUES (?, ?, ?);

-- name: ListTaskComments :many
SELECT tc.*, u.email as user_email
FROM task_comments tc
JOIN users u ON tc.user_id = u.id
WHERE tc.task_id = ?
ORDER BY tc.created_at ASC;