package server

import (
	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/training-judge-center/backend/internal/config"
	"github.com/training-judge-center/backend/internal/server/handler"
	"github.com/training-judge-center/backend/internal/server/middleware"
)

<<<<<<< Updated upstream
func NewRouter() *chi.Mux {
=======
type Handlers struct {
	User *handler.UserHandler
	Auth *handler.AuthHandler
}

func NewRouter(cfg *config.Config, h *Handlers) *chi.Mux {
>>>>>>> Stashed changes
	r := chi.NewRouter()

	r.Use(chimw.Logger)
	r.Use(chimw.Recoverer)
	r.Use(chimw.RequestID)
	r.Use(middleware.CORS(cfg.FrontendURL))

	healthHandler := handler.NewHealthHandler()

<<<<<<< Updated upstream
	r.Get("/ping", healthHandler.Ping)
=======
	r.Route("/auth", func(r chi.Router) {
		r.Post("/google", h.Auth.GoogleLogin)
	})

	r.Route("/users", func(r chi.Router) {
		r.Post("/", h.User.Create)
	})
>>>>>>> Stashed changes

	return r
}
