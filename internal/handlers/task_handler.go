package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/egor_lukyanovich/moon_test_application/internal/db"
	json_resp "github.com/egor_lukyanovich/moon_test_application/pkg/json"
	id_helper "github.com/egor_lukyanovich/moon_test_application/pkg/routing"
	"github.com/go-chi/chi/v5"
	"github.com/redis/go-redis/v9"
)

type TaskHandlers struct {
	q     *db.Queries
	db    *sql.DB
	redis *redis.Client
}

func NewTaskHandlers(q *db.Queries, database *sql.DB, redisClient *redis.Client) *TaskHandlers {
	return &TaskHandlers{q: q, db: database, redis: redisClient}
}

func (h *TaskHandlers) CreateTask(w http.ResponseWriter, r *http.Request) {
	userID, ok := id_helper.GetUserIDHelper(r.Context())
	if !ok {
		json_resp.RespondError(w, 401, "UNAUTHORIZED", "unauthorized")
		return
	}

	var req struct {
		Title       string `json:"title"`
		Description string `json:"description"`
		Status      string `json:"status"`
		TeamID      int64  `json:"team_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		json_resp.RespondError(w, 400, "BAD_REQUEST", "invalid json")
		return
	}

	if !id_helper.CheckTeamRole(r.Context(), h.q, req.TeamID, userID) {
		json_resp.RespondError(w, 403, "FORBIDDEN", "you are not a member of this team")
		return
	}

	res, err := h.q.CreateTask(r.Context(), db.CreateTaskParams{
		Title:       req.Title,
		Description: sql.NullString{String: req.Description, Valid: req.Description != ""},
		Status:      db.TasksStatus(req.Status),
		TeamID:      req.TeamID,
		CreatedBy:   userID,
	})
	if err != nil {
		json_resp.RespondError(w, 500, "INTERNAL_ERROR", "failed to create task")
		return
	}

	taskID, _ := res.LastInsertId()
	json_resp.RespondJSON(w, 201, map[string]interface{}{"task_id": taskID})
}

func (h *TaskHandlers) ListTasks(w http.ResponseWriter, r *http.Request) {
	teamID, _ := strconv.ParseInt(r.URL.Query().Get("team_id"), 10, 64)
	if teamID == 0 {
		json_resp.RespondError(w, 400, "BAD_REQUEST", "team_id is required")
		return
	}

	status := r.URL.Query().Get("status")
	assigneeStr := r.URL.Query().Get("assignee_id")

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	limit := 10
	offset := (page - 1) * limit

	cacheKey := fmt.Sprintf("tasks:t:%d:s:%s:a:%s:p:%d", teamID, status, assigneeStr, page)
	cachedData, err := h.redis.Get(r.Context(), cacheKey).Result()
	if err == nil {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(cachedData))
		return
	}

	statusNull := db.NullTasksStatus{}
	if status != "" {
		statusNull = db.NullTasksStatus{
			TasksStatus: db.TasksStatus(status),
			Valid:       true,
		}
	}

	var assigneeNull sql.NullInt64
	if assigneeStr != "" {
		aID, _ := strconv.ParseInt(assigneeStr, 10, 64)
		assigneeNull = sql.NullInt64{Int64: aID, Valid: true}
	}

	tasks, err := h.q.ListTasks(r.Context(), db.ListTasksParams{
		TeamID:     teamID,
		Status:     statusNull,
		AssigneeID: assigneeNull,
		Limit:      int32(limit),
		Offset:     int32(offset),
	})
	if err != nil {
		json_resp.RespondError(w, 500, "INTERNAL_ERROR", "failed to fetch tasks")
		return
	}
	if tasks == nil {
		tasks = []db.Task{}
	}

	dataToCache, _ := json.Marshal(tasks)
	h.redis.Set(r.Context(), cacheKey, dataToCache, 5*time.Minute)

	json_resp.RespondJSON(w, 200, tasks)
}

func (h *TaskHandlers) UpdateTask(w http.ResponseWriter, r *http.Request) {
	userID, ok := id_helper.GetUserIDHelper(r.Context())
	if !ok {
		return
	}

	taskID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		json_resp.RespondError(w, 400, "BAD_REQUEST", "invalid task id")
		return
	}

	var req struct {
		Title      string `json:"title"`
		Status     string `json:"status"`
		AssigneeID *int64 `json:"assignee_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		json_resp.RespondError(w, 400, "BAD_REQUEST", "invalid json")
		return
	}

	tx, err := h.db.BeginTx(r.Context(), nil)
	if err != nil {
		json_resp.RespondError(w, 500, "INTERNAL_ERROR", "tx failed")
		return
	}
	defer tx.Rollback()
	qtx := h.q.WithTx(tx)

	oldTask, err := qtx.GetTaskByID(r.Context(), taskID)
	if err != nil {
		json_resp.RespondError(w, 404, "NOT_FOUND", "task not found")
		return
	}

	if !id_helper.CheckTeamRole(r.Context(), qtx, oldTask.TeamID, userID) {
		json_resp.RespondError(w, 403, "FORBIDDEN", "access denied")
		return
	}

	var newAssignee sql.NullInt64
	if req.AssigneeID != nil {
		newAssignee = sql.NullInt64{Int64: *req.AssigneeID, Valid: true}
	}
	err = qtx.UpdateTask(r.Context(), db.UpdateTaskParams{
		ID:          taskID,
		Title:       req.Title,
		Status:      db.TasksStatus(req.Status),
		AssigneeID:  newAssignee,
		Description: oldTask.Description,
	})
	if err != nil {
		json_resp.RespondError(w, 500, "INTERNAL_ERROR", "failed to update task")
		return
	}

	if string(oldTask.Status) != req.Status {
		_ = qtx.CreateTaskHistory(r.Context(), db.CreateTaskHistoryParams{
			TaskID:     taskID,
			ChangedBy:  sql.NullInt64{Int64: userID, Valid: true},
			ChangeType: "status_update",
			OldValue:   sql.NullString{String: string(oldTask.Status), Valid: true},
			NewValue:   sql.NullString{String: req.Status, Valid: true},
		})
	}

	tx.Commit()
	json_resp.RespondJSON(w, 200, map[string]string{"status": "updated"})
}
