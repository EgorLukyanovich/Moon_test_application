-- name: GetTeamStats :many
SELECT 
    t.name AS team_name,
    COUNT(DISTINCT tm.user_id) AS members_count,
    COUNT(DISTINCT task.id) AS done_tasks_last_7_days
FROM teams t
LEFT JOIN team_members tm ON t.id = tm.team_id
LEFT JOIN tasks task ON t.id = task.team_id 
    AND task.status = 'done' 
    AND task.updated_at >= NOW() - INTERVAL 7 DAY
GROUP BY t.id, t.name;

-- name: GetTopUsersPerTeam :many
SELECT 
    final_tab.team_id,
    final_tab.user_id,
    final_tab.task_count,
    final_tab.rank_num
FROM (
    SELECT 
        user_counts.team_id,
        user_counts.user_id,
        user_counts.task_count,
        DENSE_RANK() OVER (PARTITION BY user_counts.team_id ORDER BY user_counts.task_count DESC) as rank_num
    FROM (
        SELECT 
            team_id,
            created_by AS user_id,
            COUNT(*) AS task_count
        FROM tasks
        WHERE created_at >= NOW() - INTERVAL 1 MONTH
        GROUP BY team_id, created_by
    ) AS user_counts
) AS final_tab
WHERE final_tab.rank_num <= 3;

-- name: FindInvalidTasks :many
SELECT t.id, t.title, t.team_id, t.assignee_id
FROM tasks t
LEFT JOIN team_members tm ON t.team_id = tm.team_id AND t.assignee_id = tm.user_id
WHERE t.assignee_id IS NOT NULL 
  AND tm.user_id IS NULL;