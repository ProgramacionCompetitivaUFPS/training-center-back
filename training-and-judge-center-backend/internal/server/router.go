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
		r.Post("/users/email-change/request", h.User.RequestEmailChange)
		r.Post("/users/email-change/confirm", h.User.ConfirmEmailChange)
	})

	// Protected routes — admin only
	r.Route("/admin", func(r chi.Router) {
		r.Use(middleware.Auth(s.TokenService))
		r.Use(middleware.RequireRole(user.RoleAdmin))

		r.Get("/users", h.User.ListUsers)
		r.Put("/users/{id}", h.User.AdminUpdateUser)
		r.Post("/users/{id}/deactivate", h.User.AdminDeactivateUser)
	})

	return r
}

