package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/egor_lukyanovich/moon_test_application/internal/db"
	json_resp "github.com/egor_lukyanovich/moon_test_application/pkg/json"
	id_helper "github.com/egor_lukyanovich/moon_test_application/pkg/routing"
	"github.com/go-chi/chi/v5"
)

type TeamHandlers struct {
	q  *db.Queries
	db *sql.DB
}

func NewTeamHandlers(q *db.Queries, database *sql.DB) *TeamHandlers {
	return &TeamHandlers{q: q, db: database}
}

func (h *TeamHandlers) CreateTeam(w http.ResponseWriter, r *http.Request) {
	userID, ok := id_helper.GetUserIDHelper(r.Context())
	if !ok {
		json_resp.RespondError(w, 401, "UNAUTHORIZED", "unauthorized")
		return
	}

	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" {
		json_resp.RespondError(w, 400, "BAD_REQUEST", "invalid json or empty name")
		return
	}

	tx, err := h.db.BeginTx(r.Context(), nil)
	if err != nil {
		json_resp.RespondError(w, 500, "INTERNAL_ERROR", "failed to start tx")
		return
	}
	defer tx.Rollback()

	qtx := h.q.WithTx(tx)

	res, err := qtx.CreateTeam(r.Context(), db.CreateTeamParams{
		Name:      req.Name,
		CreatedBy: userID,
	})
	if err != nil {
		json_resp.RespondError(w, 500, "INTERNAL_ERROR", "failed to create team")
		return
	}

	teamID, _ := res.LastInsertId()

	err = qtx.AddTeamMember(r.Context(), db.AddTeamMemberParams{
		TeamID: teamID,
		UserID: userID,
		Role:   db.TeamMembersRoleOwner,
	})
	if err != nil {
		json_resp.RespondError(w, 500, "INTERNAL_ERROR", "failed to add team owner")
		return
	}

	if err := tx.Commit(); err != nil {
		json_resp.RespondError(w, 500, "INTERNAL_ERROR", "failed to commit tx")
		return
	}

	json_resp.RespondJSON(w, 201, map[string]interface{}{
		"team_id": teamID,
	})
}

func (h *TeamHandlers) ListTeams(w http.ResponseWriter, r *http.Request) {
	userID, ok := id_helper.GetUserIDHelper(r.Context())
	if !ok {
		json_resp.RespondError(w, 401, "UNAUTHORIZED", "unauthorized")
		return
	}

	teams, err := h.q.ListUserTeams(r.Context(), userID)
	if err != nil {
		json_resp.RespondError(w, 500, "INTERNAL_ERROR", "failed to get teams")
		return
	}
	if teams == nil {
		teams = []db.ListUserTeamsRow{}
	}

	json_resp.RespondJSON(w, 200, teams)
}

func (h *TeamHandlers) InviteToTeam(w http.ResponseWriter, r *http.Request) {
	inviterID, ok := id_helper.GetUserIDHelper(r.Context())
	if !ok {
		json_resp.RespondError(w, 401, "UNAUTHORIZED", "unauthorized")
		return
	}

	teamID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		json_resp.RespondError(w, 400, "BAD_REQUEST", "invalid team id")
		return
	}

	if !id_helper.CheckTeamRole(r.Context(), h.q, teamID, inviterID, "owner", "admin") {
		json_resp.RespondError(w, 403, "FORBIDDEN", "only owner or admin can invite")
		return
	}

	var req struct {
		UserID int64  `json:"user_id"`
		Role   string `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		json_resp.RespondError(w, 400, "BAD_REQUEST", "invalid json")
		return
	}

	err = h.q.AddTeamMember(r.Context(), db.AddTeamMemberParams{
		TeamID: teamID,
		UserID: req.UserID,
		Role:   db.TeamMembersRole(req.Role),
	})
	if err != nil {
		json_resp.RespondError(w, 409, "CONFLICT", "user already in team or not found")
		return
	}

	json_resp.RespondJSON(w, 200, map[string]string{"status": "invited"})
}
