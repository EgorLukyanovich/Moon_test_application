package main

import (
	"log"

	"github.com/egor_lukyanovich/moon_test_application/pkg/app"
)

func main() {
	//ctx := context.Background()
	storage, _, err := app.InitDB()
	if err != nil {
		log.Fatalf("DB initialization failed: %v", err)
	}

	defer storage.DB.Close()
}
