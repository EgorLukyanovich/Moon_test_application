package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/egor_lukyanovich/moon_test_application/internal/handlers"
	"github.com/egor_lukyanovich/moon_test_application/pkg/app"
	"github.com/egor_lukyanovich/moon_test_application/pkg/routing"
	"github.com/go-chi/chi/v5"
)

func main() {
	storage, port, err := app.InitDB()
	if err != nil {
		log.Fatalf("DB initialization failed: %v", err)
	}

	defer func() {
		storage.DB.Close()
		storage.Redis.Close()
		log.Println("Database and Redis connections closed")
	}()

	r := routing.NewRouter()

	authH := routing.NewAuthHandlers(storage.Queries)
	teamH := handlers.NewTeamHandlers(storage.Queries, storage.DB)
	taskH := handlers.NewTaskHandlers(storage.Queries, storage.DB, storage.Redis)
	historyH := handlers.NewHistoryHandlers(storage.Queries)
	statsH := handlers.NewStatsHandlers(storage.Queries)

	r.Route("/api/v1", func(api chi.Router) {
		api.Group(func(public chi.Router) {
			public.Post("/register", authH.Register)
			public.Post("/login", authH.Login)
		})

		api.Group(func(protected chi.Router) {
			protected.Use(routing.AuthMiddleware)

			protected.Post("/teams", teamH.CreateTeam)
			protected.Get("/teams", teamH.ListTeams)
			protected.Post("/teams/{id}/invite", teamH.InviteToTeam)

			protected.Post("/tasks", taskH.CreateTask)
			protected.Get("/tasks", taskH.ListTasks)
			protected.Put("/tasks/{id}", taskH.UpdateTask)

			protected.Get("/tasks/{id}/history", historyH.GetTaskHistory)

			protected.Get("/stats/teams", statsH.GetTeamStats)
			protected.Get("/stats/top-users", statsH.GetTopUsers)
			protected.Get("/stats/invalid-tasks", statsH.GetInvalidTasks)
		})
	})

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: r,
	}

	go func() {
		log.Printf("Server starting on port %s", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exiting gracefully")
}
