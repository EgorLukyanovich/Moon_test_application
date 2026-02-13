package routing

import (
	"log"
	"net/http"

	// Укажи свой путь до пакета app

	"github.com/egor_lukyanovich/moon_test_application/pkg/app"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func NewRouter(storage *app.Storage) *chi.Mux {
	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte("OK")); err != nil {
			log.Println("write response failed:", err)
		}
	})

	r.Route("/api/v1", func(r chi.Router) {

		r.Group(func(r chi.Router) {
			r.Post("/register", handleRegister(storage))
			r.Post("/login", handleLogin(storage))
		})

		r.Group(func(r chi.Router) {
			r.Use(AuthMiddleware)

			//			r.Post("/teams", handleCreateTeam(storage))
			//			r.Get("/teams", handleListTeams(storage))
			//			r.Post("/teams/{id}/invite", handleInviteToTeam(storage))

			//			r.Post("/tasks", handleCreateTask(storage))
			//			r.Get("/tasks", handleListTasks(storage))
			//			r.Put("/tasks/{id}", handleUpdateTask(storage))
			//			r.Get("/tasks/{id}/history", handleTaskHistory(storage))

			//			r.Get("/stats/teams", handleTeamStats(storage))
			//			r.Get("/stats/top-users", handleTopUsers(storage))
			//			r.Get("/stats/invalid-tasks", handleInvalidTasks(storage))
		})
	})

	return r
}
