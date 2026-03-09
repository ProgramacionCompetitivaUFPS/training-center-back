package server

import (
	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/training-judge-center/backend/internal/config"
	"github.com/training-judge-center/backend/internal/server/handler"
	"github.com/training-judge-center/backend/internal/server/middleware"
)

type Handlers struct {
	Problem *handler.ProblemHandler
}

func NewRouter(cfg *config.Config, h *Handlers) *chi.Mux {
	r := chi.NewRouter()

	r.Use(chimw.Logger)
	r.Use(chimw.Recoverer)
	r.Use(chimw.RequestID)

	if cfg.MockAuth {
		r.Use(middleware.MockAuth)
	}

	healthHandler := handler.NewHealthHandler()
	r.Get("/ping", healthHandler.Ping)

	r.Post("/problems", h.Problem.Create)

	return r
}

