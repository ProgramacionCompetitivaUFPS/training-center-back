package server

import (
	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/training-judge-center/backend/internal/config"
	"github.com/training-judge-center/backend/internal/server/handler"
	"github.com/training-judge-center/backend/internal/server/handler/problem"
	"github.com/training-judge-center/backend/internal/server/middleware"
)

type Handlers struct {
	Problem *problem.Handler
}

func NewRouter(cfg *config.Config, h *Handlers) *chi.Mux {
	r := chi.NewRouter()

	r.Use(chimw.Logger)
	r.Use(chimw.Recoverer)
	r.Use(chimw.RequestID)

	healthHandler := handler.NewHealthHandler()
	r.Get("/ping", healthHandler.Ping)

	r.Route("/problems", func(r chi.Router) {
		if cfg.MockAuth {
			r.Use(middleware.MockAuth)
		}
		r.Post("/", h.Problem.Create)

		r.Route("/p/{slug}", func(r chi.Router) {
			r.Put("/", h.Problem.Update)

			r.Route("/files", func(r chi.Router) {
				r.Post("/", h.Problem.UploadFiles)
				r.Delete("/{fileType}", h.Problem.DeleteFile)
			})

			r.Route("/modifiers", func(r chi.Router) {
				r.Post("/", h.Problem.AddModifier)
				r.Get("/", h.Problem.ListModifiers)
				r.Delete("/{userId}", h.Problem.RemoveModifier)
			})
		})
	})

	return r
}
