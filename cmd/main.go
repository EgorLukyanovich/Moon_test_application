package main

import (
	"log"
	"net/http"

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
	defer storage.DB.Close()

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

	log.Printf("Server starting on port %s", port)
	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
