package app

import (
	"context"
	"database/sql"
	"log"
	"os"
	"time"

	DB "github.com/egor_lukyanovich/moon_test_application/internal/db"
	_ "github.com/go-sql-driver/mysql"
	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
)

type Storage struct {
	Queries *DB.Queries
	DB      *sql.DB
	Redis   *redis.Client
}

func InitDB() (*Storage, string, error) {
	_ = godotenv.Load()

	dataBaseUrl := os.Getenv("DATABASE_URL")
	if dataBaseUrl == "" {
		log.Fatal("DATABASE_URL is not found in .env")
	}

	db, err := sql.Open("mysql", dataBaseUrl)
	if err != nil {
		log.Fatal("Failed to open database:", err)
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err := db.PingContext(context.Background()); err != nil {
		log.Fatal("db ping failed: ", err)
	}

	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}

	rdb := redis.NewClient(&redis.Options{
		Addr: redisAddr,
	})

	if err := rdb.Ping(context.Background()).Err(); err != nil {
		log.Println("WARNING: Redis ping failed, caching might not work:", err)
	}

	portString := os.Getenv("SERVER_PORT")
	if portString == "" {
		log.Fatal("SERVER_PORT is not found in .env")
	}

	queries := DB.New(db)

	storage := &Storage{
		Queries: queries,
		DB:      db,
		Redis:   rdb,
	}

	return storage, portString, nil
}
