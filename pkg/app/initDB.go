package app

import (
	"context"
	"database/sql"
	"log"
	"os"

	DB "github.com/egor_lukyanovich/moon_test_application/internal/db"
	_ "github.com/go-sql-driver/mysql"
	"github.com/joho/godotenv"
)

type Storage struct {
	Queries *DB.Queries
	DB      *sql.DB
}

func InitDB() (*Storage, string, error) {
	_ = godotenv.Load()

	dataBaseUrl := os.Getenv("DATABASE_URL")
	if dataBaseUrl == "" {
		log.Fatal("DataBaseUrl is not found in .env:")
	}

	db, err := sql.Open("pgx", dataBaseUrl)
	if err != nil {
		log.Fatal("Failed to open a database:", err)
	}

	portString := os.Getenv("SERVER_PORT")
	if portString == "" {
		log.Fatal("Port string is not found in .env")
	}

	if err := db.PingContext(context.Background()); err != nil {
		log.Fatal("db ping failed: ", err)
	}

	queries := DB.New(db)

	storage := &Storage{
		Queries: queries,
		DB:      db,
	}

	return storage, portString, nil
}
