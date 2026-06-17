package team

import (
	"net/http"

	"github.com/training-judge-center/backend/internal/adapter/http/handler"
	"github.com/training-judge-center/backend/internal/adapter/http/middleware"
	appTeam "github.com/training-judge-center/backend/internal/application/team"
	"github.com/training-judge-center/backend/internal/application/shared"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type Handler struct {
	createTeam        *appTeam.CreateTeamUseCase
	listMyTeams       *appTeam.ListMyTeamsUseCase
	getTeam           *appTeam.GetTeamUseCase
	inviteToTeam      *appTeam.InviteToTeamUseCase
	listMyInvitations *appTeam.ListMyInvitationsUseCase
	acceptInvitation  *appTeam.AcceptInvitationUseCase
	rejectInvitation  *appTeam.RejectInvitationUseCase
	leaveTeam         *appTeam.LeaveTeamUseCase
}

func NewHandler(
	createTeam *appTeam.CreateTeamUseCase,
	listMyTeams *appTeam.ListMyTeamsUseCase,
	getTeam *appTeam.GetTeamUseCase,
	inviteToTeam *appTeam.InviteToTeamUseCase,
	listMyInvitations *appTeam.ListMyInvitationsUseCase,
	acceptInvitation *appTeam.AcceptInvitationUseCase,
	rejectInvitation *appTeam.RejectInvitationUseCase,
	leaveTeam *appTeam.LeaveTeamUseCase,
) *Handler {
	return &Handler{
		createTeam:        createTeam,
		listMyTeams:       listMyTeams,
		getTeam:           getTeam,
		inviteToTeam:      inviteToTeam,
		listMyInvitations: listMyInvitations,
		acceptInvitation:  acceptInvitation,
		rejectInvitation:  rejectInvitation,
		leaveTeam:         leaveTeam,
	}
}

func (h *Handler) requireCurrentUser(w http.ResponseWriter, r *http.Request) (*shared.CurrentUser, bool) {
	claims := middleware.GetClaims(r.Context())
	if claims == nil {
		handler.WriteError(r.Context(), w, apperror.NewUnauthorized(apperror.ErrCodeUnauthorized, "Invalid or missing authentication token"))
		return nil, false
	}
	u := shared.CurrentUser{ID: claims.UserID, Role: claims.Role}
	return &u, true
}
