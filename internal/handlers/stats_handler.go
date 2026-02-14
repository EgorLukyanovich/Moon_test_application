package handlers

import (
	"net/http"

	"github.com/egor_lukyanovich/moon_test_application/internal/db"
	json_resp "github.com/egor_lukyanovich/moon_test_application/pkg/json"
)

type StatsHandlers struct {
	q *db.Queries
}

func NewStatsHandlers(q *db.Queries) *StatsHandlers {
	return &StatsHandlers{q: q}
}

func (h *StatsHandlers) GetTeamStats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.q.GetTeamStats(r.Context())
	if err != nil {
		json_resp.RespondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to fetch team stats")
		return
	}

	if stats == nil {
		stats = []db.GetTeamStatsRow{}
	}

	json_resp.RespondJSON(w, http.StatusOK, stats)
}

func (h *StatsHandlers) GetTopUsers(w http.ResponseWriter, r *http.Request) {
	topUsers, err := h.q.GetTopUsersPerTeam(r.Context())
	if err != nil {
		json_resp.RespondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to fetch top users")
		return
	}

	if topUsers == nil {
		topUsers = []db.GetTopUsersPerTeamRow{}
	}

	json_resp.RespondJSON(w, http.StatusOK, topUsers)
}

func (h *StatsHandlers) GetInvalidTasks(w http.ResponseWriter, r *http.Request) {
	invalidTasks, err := h.q.FindInvalidTasks(r.Context())
	if err != nil {
		json_resp.RespondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to fetch invalid tasks")
		return
	}

	if invalidTasks == nil {
		invalidTasks = []db.FindInvalidTasksRow{}
	}

	json_resp.RespondJSON(w, http.StatusOK, invalidTasks)
}
