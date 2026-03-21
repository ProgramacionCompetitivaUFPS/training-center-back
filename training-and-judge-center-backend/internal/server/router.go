package server

import (
	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/training-judge-center/backend/internal/domain/user"
	"github.com/training-judge-center/backend/internal/server/handler"
	"github.com/training-judge-center/backend/internal/server/middleware"
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

	// Protected routes — authenticated users
	r.Group(func(r chi.Router) {
		r.Use(middleware.Auth(s.TokenService))

		r.Get("/users/me", h.User.GetMyProfile)
		r.Get("/users/{nickname}", h.User.GetByNickname)
		r.Put("/users", h.User.UpdateProfile)
		r.Put("/users/password", h.User.UpdatePassword)
	})

	// Protected routes — admin only
	r.Route("/admin", func(r chi.Router) {
		r.Use(middleware.Auth(s.TokenService))
		r.Use(middleware.RequireRole(user.RoleAdmin))

		r.Put("/users/{id}", h.User.AdminUpdateUser)
	})

	return r
}

