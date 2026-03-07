package server

import (
	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/training-judge-center/backend/internal/domain/user"
	"github.com/training-judge-center/backend/internal/server/handler"
)

type Handlers struct {
	User *handler.UserHandler
	Auth *handler.AuthHandler
}

type Services struct {
	TokenService user.TokenService
}

func NewRouter(h *Handlers, s *Services) *chi.Mux {
	r := chi.NewRouter()

	r.Use(chimw.Logger)
	r.Use(chimw.Recoverer)
	r.Use(chimw.RequestID)

	healthHandler := handler.NewHealthHandler()
	r.Get("/ping", healthHandler.Ping)

	// Public routes
	r.Post("/users", h.User.Create)

	r.Route("/auth", func(r chi.Router) {
		r.Post("/login", h.Auth.Login)
	})

	return r
}
