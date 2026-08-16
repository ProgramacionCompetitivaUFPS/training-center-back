package http

import (
	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	httpSwagger "github.com/swaggo/http-swagger"
	_ "github.com/training-judge-center/backend/docs"
	handler2 "github.com/training-judge-center/backend/internal/adapter/http/handler"
	handlerContest "github.com/training-judge-center/backend/internal/adapter/http/handler/contest"
	"github.com/training-judge-center/backend/internal/adapter/http/handler/group"
	handlerMaterial "github.com/training-judge-center/backend/internal/adapter/http/handler/material"
	"github.com/training-judge-center/backend/internal/adapter/http/handler/problem"
	handlerSubmission "github.com/training-judge-center/backend/internal/adapter/http/handler/submission"
	handlerTeam "github.com/training-judge-center/backend/internal/adapter/http/handler/team"
	handlerUser "github.com/training-judge-center/backend/internal/adapter/http/handler/user"
	"github.com/training-judge-center/backend/internal/adapter/http/middleware"
	"github.com/training-judge-center/backend/internal/domain/shared"
	"github.com/training-judge-center/backend/internal/domain/user"
)

type Handlers struct {
	Problem    *problem.Handler
	User       *handlerUser.Handler
	Auth       *handler2.AuthHandler
	Group      *group.Handler
	Material   *handlerMaterial.Handler
	Contest    *handlerContest.Handler
	Team       *handlerTeam.Handler
	Submission *handlerSubmission.Handler
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
	r.Use(chimw.RequestID)
	r.Use(chimw.Logger)
	r.Use(chimw.Recoverer)

	r.Get("/swagger/*", httpSwagger.WrapHandler)

	healthHandler := handler2.NewHealthHandler()
	r.Get("/ping", healthHandler.Ping)

	// public
	r.Post("/users", h.User.Create)
	r.Post("/password/forgot", h.User.RequestPasswordRecovery)
	r.Post("/password/reset", h.User.ResetPassword)
	r.Route("/auth", func(r chi.Router) {
		r.Post("/login", h.Auth.Login)
		r.Post("/google", h.Auth.LoginWithGoogle)
		r.Post("/refresh", h.Auth.Refresh)
		r.Post("/logout", h.Auth.Logout)
	})

	// authenticated
	r.Group(func(r chi.Router) {
		r.Use(middleware.Auth(s.TokenService, s.SessionInvalidator))

		r.Route("/groups", func(r chi.Router) {
			r.Post("/", h.Group.Create)
			r.Get("/", h.Group.ListGroups)
			r.Get("/{groupId}", h.Group.GetGroup)
			r.Patch("/{groupId}", h.Group.UpdateGroup)
			r.Delete("/{groupId}", h.Group.DeleteGroup)
			r.Post("/{groupId}/join", h.Group.Join)

			r.Route("/{groupId}/invitations", func(r chi.Router) {
				r.Post("/", h.Group.GenerateInvite)
				r.Get("/", h.Group.ListGroupInvitations)
				r.Delete("/{invitationId}", h.Group.RevokeInvitation)
				r.Post("/accept", h.Group.AcceptInvite)
				r.Post("/targeted", h.Group.InviteByNicknames)
			})

			r.Route("/{groupId}/members", func(r chi.Router) {
				r.Post("/", h.Group.AddMember)
				r.Get("/", h.Group.ListMembers)
				r.Delete("/me", h.Group.LeaveGroup)
				r.Delete("/{nickname}", h.Group.RemoveMember)
				r.Patch("/{nickname}", h.Group.ChangeRole)
			})

			r.Route("/{groupId}/requests", func(r chi.Router) {
				r.Post("/", h.Group.RequestJoin)
				r.Get("/", h.Group.ListJoinRequests)
				r.Get("/me", h.Group.GetMyRequest)
				r.Delete("/me", h.Group.CancelMyRequest)
				r.Patch("/{requestId}", h.Group.UpdateJoinRequest)
			})

			r.Route("/{groupId}/contests", func(r chi.Router) {
				r.Post("/", h.Contest.Create)
				r.Get("/", h.Contest.List)
				r.Get("/{contestId}", h.Contest.Get)
				r.Put("/{contestId}", h.Contest.Update)
				r.Delete("/{contestId}", h.Contest.Delete)
				r.Post("/{contestId}/register", h.Contest.Register)
				r.Delete("/{contestId}/register", h.Contest.Unregister)
				r.Get("/{contestId}/register/status", h.Contest.GetRegistrationStatus)
				r.Get("/{contestId}/registrations", h.Contest.ListRegistrations)
				r.Get("/{contestId}/standings", h.Contest.GetStandings)
				r.Get("/{contestId}/submissions", h.Contest.ListContestSubmissions)
				r.Route("/{contestId}/team-registrations", func(r chi.Router) {
					r.Get("/", h.Team.ListTeamRegistrations)
					r.Post("/{teamId}", h.Team.RegisterTeamToContest)
					r.Put("/{teamId}", h.Team.UpdateTeamRegistration)
					r.Delete("/{teamId}", h.Team.UnregisterTeamFromContest)
				})

				r.Post("/{contestId}/problems/{problemSlug}/submissions", h.Submission.SubmitContest)
				r.Post("/{contestId}/problems/{problemSlug}/rejudge", h.Problem.RejudgeContest)
			})

			r.Route("/{groupId}/materials", func(r chi.Router) {
				r.Post("/", h.Material.Create)
				r.Get("/", h.Material.List)
				r.Get("/{materialId}", h.Material.Get)
				r.Patch("/{materialId}", h.Material.Update)
				r.Delete("/{materialId}", h.Material.Delete)
				r.Post("/{materialId}/publish", h.Material.Publish)
				r.Post("/{materialId}/unpublish", h.Material.Unpublish)
				r.Post("/{materialId}/pin", h.Material.Pin)
				r.Post("/{materialId}/unpin", h.Material.Unpin)
			})
		})

		r.Get("/contests", h.Contest.ListMyContests)

		r.Route("/submissions", func(r chi.Router) {
			r.Get("/{submissionId}", h.Submission.GetSubmission)
			r.Patch("/{submissionId}/visibility", h.Submission.UpdateVisibility)
			r.Post("/{submissionId}/rejudge", h.Submission.RejudgeSubmission)
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
				r.Post("/rejudge", h.Problem.Rejudge)
				r.Patch("/accessibility", h.Problem.ChangeAccessibility)

				r.Get("/statistics", h.Problem.GetStatistics)

				r.Get("/submissions", h.Submission.ListProblemSubmissions)
				r.Post("/submissions", h.Submission.Submit)

				r.Route("/files", func(r chi.Router) {
					r.Post("/", h.Problem.UploadFiles)
					r.Delete("/{fileType}", h.Problem.DeleteFile)
				})

				r.Route("/modifiers", func(r chi.Router) {
					r.Post("/", h.Problem.AddModifier)
					r.Get("/", h.Problem.ListModifiers)
					r.Delete("/{nickname}", h.Problem.RemoveModifier)
				})
			})
		})

		r.Route("/teams", func(r chi.Router) {
			r.Post("/", h.Team.Create)
			r.Get("/{teamId}", h.Team.GetTeam)
			r.Route("/{teamId}/invitations", func(r chi.Router) {
				r.Post("/", h.Team.InviteToTeam)
			})
			r.Delete("/{teamId}/members/me", h.Team.LeaveTeam)
		})

		r.Route("/team-invitations", func(r chi.Router) {
			r.Post("/{invitationId}/accept", h.Team.AcceptInvitation)
			r.Delete("/{invitationId}", h.Team.RejectInvitation)
		})

		r.Get("/users/me", h.User.GetMyProfile)
		r.Get("/users/me/dashboard", h.User.GetDashboard)
		r.Get("/users/me/stats", h.User.GetStats)
		r.Get("/users/me/groups", h.Group.ListMyGroups)
		r.Get("/users/me/teams", h.Team.ListMyTeams)
		r.Get("/users/me/team-invitations", h.Team.ListMyInvitations)
		r.Get("/users/me/submissions", h.Submission.ListMySubmissions)
		r.Get("/users/{nickname}", h.User.GetByNickname)
		r.Put("/users", h.User.UpdateProfile)
		r.Put("/users/password", h.User.UpdatePassword)
		r.Post("/users/email-change/request", h.User.RequestEmailChange)
		r.Post("/users/email-change/confirm", h.User.ConfirmEmailChange)
		r.Post("/users/deactivation", h.User.RequestDeactivation)
		r.Post("/users/deactivation/confirm", h.User.ConfirmDeactivation)
		r.Post("/users/google", h.User.LinkGoogle)
		r.Delete("/users/google", h.User.UnlinkGoogle)
	})

	// Protected routes — admin only
	r.Route("/admin", func(r chi.Router) {
		r.Use(middleware.Auth(s.TokenService, s.SessionInvalidator))
		r.Use(middleware.RequireRole(shared.RoleAdmin))

		r.Get("/users", h.User.ListUsers)
		r.Put("/users/{id}", h.User.AdminUpdateUser)
		r.Post("/users/{id}/deactivate", h.User.AdminDeactivateUser)

		r.Post("/problems/{slug}/rejudge", h.Problem.AdminRejudge)
		r.Post("/submissions/{submissionId}/rejudge", h.Submission.AdminRejudgeSubmission)
	})

	return r
}
