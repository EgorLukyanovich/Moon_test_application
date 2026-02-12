package main

import (
	"context"
	"log"
)

func main() {
	ctx := context.Background()
	storage, port, err := app.InitDB()
	if err != nil {
		log.Fatalf("DB initialization failed: %v", err)
	}

	defer storage.DB.Close()
}
