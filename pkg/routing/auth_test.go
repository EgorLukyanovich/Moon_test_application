package routing

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/egor_lukyanovich/moon_test_application/internal/db"
	_ "github.com/go-sql-driver/mysql"
	"github.com/testcontainers/testcontainers-go/modules/mysql"
)

func setupAuthDB(t *testing.T) (*sql.DB, func()) {
	ctx := context.Background()

	mysqlContainer, err := mysql.Run(ctx,
		"mysql:8.0",
		mysql.WithDatabase("testdb"),
		mysql.WithUsername("user"),
		mysql.WithPassword("pass"),
	)
	if err != nil {
		t.Fatalf("failed to start mysql: %v", err)
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

func TestRegisterAndLogin(t *testing.T) {
	database, cleanup := setupAuthDB(t)
	defer cleanup()

	queries := db.New(database)
	authHandlers := NewAuthHandlers(queries)

	reqBody := []byte(`{"email": "test@avito.ru", "password": "superpassword"}`)
	reqReg := httptest.NewRequest(http.MethodPost, "/register", bytes.NewBuffer(reqBody))
	rrReg := httptest.NewRecorder()

	authHandlers.Register(rrReg, reqReg)

	if rrReg.Code != http.StatusCreated {
		t.Errorf("expected status %v; got %v. Body: %s", http.StatusCreated, rrReg.Code, rrReg.Body.String())
	}

	user, err := queries.GetUserByEmail(context.Background(), "test@avito.ru")
	if err != nil {
		t.Fatalf("user not found in db: %v", err)
	}
	if user.PasswordHash == "superpassword" || user.PasswordHash == "" {
		t.Errorf("password was not hashed properly")
	}

	reqLogin := httptest.NewRequest(http.MethodPost, "/login", bytes.NewBuffer(reqBody))
	rrLogin := httptest.NewRecorder()

	authHandlers.Login(rrLogin, reqLogin)

	if rrLogin.Code != http.StatusOK {
		t.Errorf("expected status %v for valid login; got %v", http.StatusOK, rrLogin.Code)
	}

	var response map[string]string
	json.NewDecoder(rrLogin.Body).Decode(&response)

	if response["token"] == "" {
		t.Errorf("expected JWT token in response, got empty")
	}

	badReqBody := []byte(`{"email": "test@avito.ru", "password": "wrongpassword"}`)
	reqBadLogin := httptest.NewRequest(http.MethodPost, "/login", bytes.NewBuffer(badReqBody))
	rrBadLogin := httptest.NewRecorder()

	authHandlers.Login(rrBadLogin, reqBadLogin)

	if rrBadLogin.Code != http.StatusUnauthorized {
		t.Errorf("expected status %v for invalid login; got %v", http.StatusUnauthorized, rrBadLogin.Code)
	}
}
