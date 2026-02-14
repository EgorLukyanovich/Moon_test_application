package handlers

import (
	"net/http"
	"strconv"

	"github.com/egor_lukyanovich/moon_test_application/internal/db"
	json_resp "github.com/egor_lukyanovich/moon_test_application/pkg/json"
	id_helper "github.com/egor_lukyanovich/moon_test_application/pkg/routing"
	"github.com/go-chi/chi/v5"
)

type HistoryHandlers struct {
	q *db.Queries
}

func NewHistoryHandlers(q *db.Queries) *HistoryHandlers {
	return &HistoryHandlers{q: q}
}

func (h *HistoryHandlers) GetTaskHistory(w http.ResponseWriter, r *http.Request) {
	userID, ok := id_helper.GetUserIDHelper(r.Context())
	if !ok {
		json_resp.RespondError(w, http.StatusUnauthorized, "UNAUTHORIZED", "unauthorized")
		return
	}

	taskID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		json_resp.RespondError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid task id")
		return
	}

	task, err := h.q.GetTaskByID(r.Context(), taskID)
	if err != nil {
		json_resp.RespondError(w, http.StatusNotFound, "NOT_FOUND", "task not found")
		return
	}

	if !id_helper.CheckTeamRole(r.Context(), h.q, task.TeamID, userID) {
		json_resp.RespondError(w, http.StatusForbidden, "FORBIDDEN", "you don't have access to this task's history")
		return
	}

	history, err := h.q.ListTaskHistory(r.Context(), taskID)
	if err != nil {
		json_resp.RespondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to fetch task history")
		return
	}

	if history == nil {
		history = []db.ListTaskHistoryRow{}
	}

	json_resp.RespondJSON(w, http.StatusOK, history)
}
