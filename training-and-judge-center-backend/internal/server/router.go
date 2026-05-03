package server

import (
	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	httpSwagger "github.com/swaggo/http-swagger"
	_ "github.com/training-judge-center/backend/docs"
	"github.com/training-judge-center/backend/internal/domain/user"
	"github.com/training-judge-center/backend/internal/server/handler"
	"github.com/training-judge-center/backend/internal/server/handler/group"
	handlerMaterial "github.com/training-judge-center/backend/internal/server/handler/material"
	"github.com/training-judge-center/backend/internal/server/handler/problem"
	handlerUser "github.com/training-judge-center/backend/internal/server/handler/user"
	"github.com/training-judge-center/backend/internal/server/middleware"
)

type Handlers struct {
	Problem  *problem.Handler
	User     *handlerUser.UserHandler
	Auth     *handler.AuthHandler
	Group    *group.Handler
	Material *handlerMaterial.Handler
}

type Services struct {
	TokenService       user.TokenService
	SessionInvalidator user.SessionInvalidator
}

func NewRouter(h *Handlers, s *Services, allowedOrigins []string) *chi.Mux {
	r := chi.NewRouter()

	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   allowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: true,
		MaxAge:           300,
	}))
	r.Use(chimw.Logger)
	r.Use(chimw.Recoverer)
	r.Use(chimw.RequestID)

	r.Get("/swagger/*", httpSwagger.WrapHandler)

	healthHandler := handler.NewHealthHandler()
	r.Get("/ping", healthHandler.Ping)

	// Group and Problem routes — require authentication
	r.Group(func(r chi.Router) {
		r.Use(middleware.Auth(s.TokenService, s.SessionInvalidator))

		r.Route("/groups", func(r chi.Router) {
			r.Post("/", h.Group.Create)
			r.Get("/", h.Group.ListGroups)
			r.Get("/{groupId}", h.Group.GetGroup)
			r.Post("/{groupId}/join", h.Group.Join)

			r.Route("/{groupId}/invitations", func(r chi.Router) {
				r.Post("/", h.Group.GenerateInvite)
				r.Post("/accept", h.Group.AcceptInvite)
			})

			r.Route("/{groupId}/requests", func(r chi.Router) {
				r.Post("/", h.Group.RequestJoin)
				r.Get("/", h.Group.ListJoinRequests)
				r.Get("/me", h.Group.GetMyRequest)
				r.Delete("/me", h.Group.CancelMyRequest)
				r.Patch("/{requestId}", h.Group.UpdateJoinRequest)
			})

			r.Route("/{groupId}/materials", func(r chi.Router) {
				r.Post("/", h.Material.Create)
				r.Get("/", h.Material.List)
				r.Get("/{materialId}", h.Material.Get)
				r.Patch("/{materialId}", h.Material.Update)
				r.Post("/{materialId}/publish", h.Material.Publish)
				r.Post("/{materialId}/unpublish", h.Material.Unpublish)
				r.Post("/{materialId}/pin", h.Material.Pin)
				r.Post("/{materialId}/unpin", h.Material.Unpin)
			})
		})

		r.Route("/problems", func(r chi.Router) {
			r.Get("/", h.Problem.ListProblems)
			r.Post("/", h.Problem.Create)
			r.Post("/import", h.Problem.Import)

			r.Route("/p/{slug}", func(r chi.Router) {
				r.Get("/", h.Problem.GetProblem)
				r.Put("/", h.Problem.Update)
				r.Delete("/", h.Problem.DeleteProblem)
				r.Post("/unpublish", h.Problem.Unpublish)
				r.Patch("/accessibility", h.Problem.ChangeAccessibility)

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
	})

	// Public user routes
	r.Post("/users", h.User.Create)
	r.Post("/password/forgot", h.User.RequestPasswordRecovery)
	r.Post("/password/reset", h.User.ResetPassword)

	r.Route("/auth", func(r chi.Router) {
		r.Post("/login", h.Auth.Login)
	})

	// Protected routes — authenticated users
	r.Group(func(r chi.Router) {
		r.Use(middleware.Auth(s.TokenService, s.SessionInvalidator))

		r.Get("/users/me", h.User.GetMyProfile)
		r.Get("/users/me/groups", h.Group.ListMyGroups)
		r.Get("/users/{nickname}", h.User.GetByNickname)
		r.Put("/users", h.User.UpdateProfile)
		r.Put("/users/password", h.User.UpdatePassword)
		r.Post("/users/email-change/request", h.User.RequestEmailChange)
		r.Post("/users/email-change/confirm", h.User.ConfirmEmailChange)
		r.Post("/users/deactivation", h.User.RequestDeactivation)
		r.Post("/users/deactivation/confirm", h.User.ConfirmDeactivation)
	})

	// Protected routes — admin only
	r.Route("/admin", func(r chi.Router) {
		r.Use(middleware.Auth(s.TokenService, s.SessionInvalidator))
		r.Use(middleware.RequireRole(user.RoleAdmin))

		r.Get("/users", h.User.ListUsers)
		r.Put("/users/{id}", h.User.AdminUpdateUser)
		r.Post("/users/{id}/deactivate", h.User.AdminDeactivateUser)
	})

	return r
}
