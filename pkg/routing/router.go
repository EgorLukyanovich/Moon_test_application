package routing

// import (
// 	"log"
// 	"net/http"

// 	"github.com/egor_lukyanovich/moon_test_application/pkg/app"
// 	"github.com/go-chi/chi/middleware"
// 	"github.com/go-chi/chi/v5"
// 	"github.com/go-chi/cors"
// )

// func NewRouter(storage app.Storage) *chi.Mux {
// 	r := chi.NewRouter()

// 	corsHandler := cors.New(cors.Options{
// 		AllowedOrigins:   []string{"http://localhost:3001", "http://localhost:3000"}, //ЕСЛИ ЧТО ПОРТ ФРОНТА ПЕРВЫЙ
// 		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
// 		AllowedHeaders:   []string{"Accept", "Content-Type", "token", "Authorization"},
// 		ExposedHeaders:   []string{"Link"},
// 		AllowCredentials: true,
// 		MaxAge:           300,
// 	})

// 	r.Use(corsHandler.Handler)

// 	r.Use(middleware.Logger)
// 	r.Use(middleware.Recoverer)

// 	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
// 		if _, err := w.Write([]byte("OK")); err != nil {
// 			log.Println("write response failed:", err)
// 		}
// 	})

// 	return r
// }
