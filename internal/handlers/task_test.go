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
	goredis "github.com/redis/go-redis/v9"
	"github.com/testcontainers/testcontainers-go/modules/mysql"
	tc_redis "github.com/testcontainers/testcontainers-go/modules/redis"
)

func setupTestDBWithTasks(t *testing.T) (*sql.DB, func()) {
	ctx := context.Background()

	mysqlContainer, err := mysql.Run(ctx,
		"mysql:8.0",
		mysql.WithDatabase("testdb"),
		mysql.WithUsername("user"),
		mysql.WithPassword("pass"),
	)
	if err != nil {
		t.Fatalf("failed to start mysql container: %v", err)
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
	);
	CREATE TABLE tasks (
		id BIGINT AUTO_INCREMENT PRIMARY KEY,
		title VARCHAR(255) NOT NULL,
		description TEXT,
		status ENUM('todo', 'in_progress', 'done') NOT NULL DEFAULT 'todo',
		team_id BIGINT NOT NULL,
		assignee_id BIGINT,
		created_by BIGINT NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
	);
	CREATE TABLE task_history (
		id BIGINT AUTO_INCREMENT PRIMARY KEY,
		task_id BIGINT NOT NULL,
		changed_by BIGINT,
		change_type VARCHAR(50) NOT NULL,
		old_value VARCHAR(255),
		new_value VARCHAR(255),
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
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

func setupTestRedis(t *testing.T) (*goredis.Client, func()) {
	ctx := context.Background()

	redisContainer, err := tc_redis.Run(ctx, "redis:7")
	if err != nil {
		t.Fatalf("failed to start redis container: %v", err)
	}

	uri, err := redisContainer.ConnectionString(ctx)
	if err != nil {
		t.Fatalf("failed to get redis uri: %v", err)
	}

	client := goredis.NewClient(&goredis.Options{
		Addr: uri[8:],
	})

	cleanup := func() {
		client.Close()
		redisContainer.Terminate(ctx)
	}

	return client, cleanup
}

func TestCreateTask(t *testing.T) {
	database, cleanupDB := setupTestDBWithTasks(t)
	defer cleanupDB()
	rdb, cleanupRedis := setupTestRedis(t)
	defer cleanupRedis()

	queries := db.New(database)
	taskHandlers := NewTaskHandlers(queries, database, rdb)

	resUser, _ := queries.CreateUser(context.Background(), db.CreateUserParams{
		Email: "task_creator@example.com", PasswordHash: "hash",
	})
	userID, _ := resUser.LastInsertId()

	resTeam, _ := queries.CreateTeam(context.Background(), db.CreateTeamParams{
		Name: "Task Team", CreatedBy: userID,
	})
	teamID, _ := resTeam.LastInsertId()

	_ = queries.AddTeamMember(context.Background(), db.AddTeamMemberParams{
		TeamID: teamID, UserID: userID, Role: "owner",
	})

	reqBody := []byte(`{"title": "Test Task", "status": "todo", "team_id": ` + strconv.FormatInt(teamID, 10) + `}`)
	req := httptest.NewRequest(http.MethodPost, "/tasks", bytes.NewBuffer(reqBody))
	req = req.WithContext(id_helper.WithUserID(req.Context(), userID))

	rr := httptest.NewRecorder()
	taskHandlers.CreateTask(rr, req)

	if rr.Code != http.StatusCreated {
		t.Errorf("expected status %v; got %v. Body: %s", http.StatusCreated, rr.Code, rr.Body.String())
	}
}

func TestListTasksAndCache(t *testing.T) {
	database, cleanupDB := setupTestDBWithTasks(t)
	defer cleanupDB()
	rdb, cleanupRedis := setupTestRedis(t)
	defer cleanupRedis()

	queries := db.New(database)
	taskHandlers := NewTaskHandlers(queries, database, rdb)

	resUser, _ := queries.CreateUser(context.Background(), db.CreateUserParams{
		Email: "list_task@example.com", PasswordHash: "hash",
	})
	userID, _ := resUser.LastInsertId()

	resTeam, _ := queries.CreateTeam(context.Background(), db.CreateTeamParams{
		Name: "List Team", CreatedBy: userID,
	})
	teamID, _ := resTeam.LastInsertId()

	_, _ = queries.CreateTask(context.Background(), db.CreateTaskParams{
		Title: "Cache me", Status: "todo", TeamID: teamID, CreatedBy: userID,
	})

	url := "/tasks?team_id=" + strconv.FormatInt(teamID, 10) + "&status=todo"
	req := httptest.NewRequest(http.MethodGet, url, nil)

	rr1 := httptest.NewRecorder()
	taskHandlers.ListTasks(rr1, req)

	if rr1.Code != http.StatusOK {
		t.Errorf("expected 200, got %v", rr1.Code)
	}

	keys, _ := rdb.Keys(context.Background(), "tasks:t:*").Result()
	if len(keys) == 0 {
		t.Errorf("expected redis to cache the response, but keys are empty")
	}

	_, _ = database.Exec("DELETE FROM tasks")

	rr2 := httptest.NewRecorder()
	taskHandlers.ListTasks(rr2, req)

	var response []map[string]interface{}
	json.NewDecoder(rr2.Body).Decode(&response)
	if len(response) == 0 {
		t.Errorf("expected data to be returned from redis cache, but got empty array")
	}
}

func TestUpdateTaskHistory(t *testing.T) {
	database, cleanupDB := setupTestDBWithTasks(t)
	defer cleanupDB()
	rdb, cleanupRedis := setupTestRedis(t)
	defer cleanupRedis()

	queries := db.New(database)
	taskHandlers := NewTaskHandlers(queries, database, rdb)

	resUser, _ := queries.CreateUser(context.Background(), db.CreateUserParams{
		Email: "updater@example.com", PasswordHash: "hash",
	})
	userID, _ := resUser.LastInsertId()

	resTeam, _ := queries.CreateTeam(context.Background(), db.CreateTeamParams{
		Name: "Update Team", CreatedBy: userID,
	})
	teamID, _ := resTeam.LastInsertId()

	_ = queries.AddTeamMember(context.Background(), db.AddTeamMemberParams{
		TeamID: teamID, UserID: userID, Role: "member",
	})

	resTask, _ := queries.CreateTask(context.Background(), db.CreateTaskParams{
		Title: "Update me", Status: "todo", TeamID: teamID, CreatedBy: userID,
	})
	taskID, _ := resTask.LastInsertId()

	reqBody := []byte(`{"title": "Updated", "status": "in_progress"}`)
	r := chi.NewRouter()
	r.Put("/tasks/{id}", taskHandlers.UpdateTask)

	url := "/tasks/" + strconv.FormatInt(taskID, 10)
	req := httptest.NewRequest(http.MethodPut, url, bytes.NewBuffer(reqBody))
	req = req.WithContext(id_helper.WithUserID(req.Context(), userID))

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %v", rr.Code)
	}

	var count int
	err := database.QueryRow("SELECT COUNT(*) FROM task_history WHERE task_id = ?", taskID).Scan(&count)
	if err != nil || count == 0 {
		t.Errorf("expected task history to be created, count: %v", count)
	}
}
