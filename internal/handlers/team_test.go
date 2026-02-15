package handlers

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/egor_lukyanovich/moon_test_application/internal/db"
	id_helper "github.com/egor_lukyanovich/moon_test_application/pkg/routing"
	"github.com/go-chi/chi/v5"
	_ "github.com/go-sql-driver/mysql"
	"github.com/testcontainers/testcontainers-go/modules/mysql"
)

func setupTestDB(t *testing.T) (*sql.DB, func()) {
	ctx := context.Background()

	mysqlContainer, err := mysql.Run(ctx,
		"mysql:8.0",
		mysql.WithDatabase("testdb"),
		mysql.WithUsername("user"),
		mysql.WithPassword("pass"),
	)
	if err != nil {
		t.Fatalf("failed to start container: %v", err)
	}

	connStr, err := mysqlContainer.ConnectionString(ctx)
	if err != nil {
		t.Fatalf("failed to get connection string: %v", err)
	}

	if !strings.Contains(connStr, "?") {
		connStr += "?"
	} else {
		connStr += "&"
	}
	connStr += "multiStatements=true&parseTime=true"

	database, err := sql.Open("mysql", connStr)
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}

	schema := `
	CREATE TABLE users (
		id BIGINT AUTO_INCREMENT PRIMARY KEY,
		email VARCHAR(255) NOT NULL UNIQUE,
		password_hash VARCHAR(255) NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);
	CREATE TABLE teams (
		id BIGINT AUTO_INCREMENT PRIMARY KEY,
		name VARCHAR(255) NOT NULL,
		created_by BIGINT NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);
	CREATE TABLE team_members (
		team_id BIGINT NOT NULL,
		user_id BIGINT NOT NULL,
		role ENUM('owner', 'admin', 'member') NOT NULL,
		joined_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		PRIMARY KEY (team_id, user_id)
	);`
	_, err = database.Exec(schema)
	if err != nil {
		t.Fatalf("failed to create schema: %v", err)
	}

	cleanup := func() {
		database.Close()
		mysqlContainer.Terminate(ctx)
	}

	return database, cleanup
}

func TestCreateTeam(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()

	queries := db.New(database)
	teamHandlers := NewTeamHandlers(queries, database)

	res, _ := queries.CreateUser(context.Background(), db.CreateUserParams{
		Email:        "test@example.com",
		PasswordHash: "hash",
	})
	userID, _ := res.LastInsertId()

	reqBody := []byte(`{"name": "Avengers"}`)
	req := httptest.NewRequest(http.MethodPost, "/teams", bytes.NewBuffer(reqBody))

	req = req.WithContext(id_helper.WithUserID(req.Context(), userID))

	rr := httptest.NewRecorder()

	teamHandlers.CreateTeam(rr, req)

	if rr.Code != http.StatusCreated {
		t.Errorf("expected status %v; got %v", http.StatusCreated, rr.Code)
	}

	var response map[string]any
	json.NewDecoder(rr.Body).Decode(&response)

	if response["team_id"] == nil {
		t.Errorf("expected team_id in response")
	}

	role, err := queries.GetUserRoleInTeam(context.Background(), db.GetUserRoleInTeamParams{
		TeamID: int64(response["team_id"].(float64)),
		UserID: userID,
	})

	if err != nil || string(role) != "owner" {
		t.Errorf("expected user to be owner, got role: %v, err: %v", role, err)
	}
}

func TestListTeams(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()

	queries := db.New(database)
	teamHandlers := NewTeamHandlers(queries, database)

	res, _ := queries.CreateUser(context.Background(), db.CreateUserParams{
		Email:        "list@example.com",
		PasswordHash: "hash",
	})
	userID, _ := res.LastInsertId()

	resTeam, _ := queries.CreateTeam(context.Background(), db.CreateTeamParams{
		Name:      "Test Team",
		CreatedBy: userID,
	})
	teamID, _ := resTeam.LastInsertId()

	_ = queries.AddTeamMember(context.Background(), db.AddTeamMemberParams{
		TeamID: teamID,
		UserID: userID,
		Role:   "owner",
	})

	req := httptest.NewRequest(http.MethodGet, "/teams", nil)
	req = req.WithContext(id_helper.WithUserID(req.Context(), userID))

	rr := httptest.NewRecorder()
	teamHandlers.ListTeams(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status %v; got %v", http.StatusOK, rr.Code)
	}

	var response []map[string]interface{}
	json.NewDecoder(rr.Body).Decode(&response)

	if len(response) == 0 {
		t.Errorf("expected at least one team in response")
	}
}

func TestInviteToTeam(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()

	queries := db.New(database)
	teamHandlers := NewTeamHandlers(queries, database)

	resInviter, _ := queries.CreateUser(context.Background(), db.CreateUserParams{
		Email:        "owner@example.com",
		PasswordHash: "hash",
	})
	inviterID, _ := resInviter.LastInsertId()

	resInvitee, _ := queries.CreateUser(context.Background(), db.CreateUserParams{
		Email:        "invitee@example.com",
		PasswordHash: "hash",
	})
	inviteeID, _ := resInvitee.LastInsertId()

	resTeam, _ := queries.CreateTeam(context.Background(), db.CreateTeamParams{
		Name:      "Invite Team",
		CreatedBy: inviterID,
	})
	teamID, _ := resTeam.LastInsertId()

	_ = queries.AddTeamMember(context.Background(), db.AddTeamMemberParams{
		TeamID: teamID,
		UserID: inviterID,
		Role:   "owner",
	})

	reqBody := []byte(`{"user_id": ` + strconv.FormatInt(inviteeID, 10) + `, "role": "member"}`)

	r := chi.NewRouter()
	r.Post("/teams/{id}/invite", teamHandlers.InviteToTeam)

	url := "/teams/" + strconv.FormatInt(teamID, 10) + "/invite"
	req := httptest.NewRequest(http.MethodPost, url, bytes.NewBuffer(reqBody))

	ctx := id_helper.WithUserID(req.Context(), inviterID)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status %v; got %v. Body: %s", http.StatusOK, rr.Code, rr.Body.String())
	}

	role, err := queries.GetUserRoleInTeam(context.Background(), db.GetUserRoleInTeamParams{
		TeamID: teamID,
		UserID: inviteeID,
	})
	if err != nil || string(role) != "member" {
		t.Errorf("expected invitee to be member, got role: %v, err: %v", role, err)
	}
}
