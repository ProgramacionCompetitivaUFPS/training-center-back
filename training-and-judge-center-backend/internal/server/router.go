package server

import (
	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/training-judge-center/backend/internal/server/handler"
)

type Handlers struct {
	User *handler.UserHandler
}

func NewRouter(h *Handlers) *chi.Mux {
	r := chi.NewRouter()

	r.Use(chimw.Logger)
	r.Use(chimw.Recoverer)
	r.Use(chimw.RequestID)

	healthHandler := handler.NewHealthHandler()
	r.Get("/ping", healthHandler.Ping)

	r.Route("/users", func(r chi.Router) {
		r.Post("/", h.User.Create)
	})

	return r
}
