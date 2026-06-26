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
	createTeam                *appTeam.CreateTeamUseCase
	listMyTeams               *appTeam.ListMyTeamsUseCase
	getTeam                   *appTeam.GetTeamUseCase
	inviteToTeam              *appTeam.InviteToTeamUseCase
	listMyInvitations         *appTeam.ListMyInvitationsUseCase
	acceptInvitation          *appTeam.AcceptInvitationUseCase
	rejectInvitation          *appTeam.RejectInvitationUseCase
	leaveTeam                 *appTeam.LeaveTeamUseCase
	registerTeamToContest     *appTeam.RegisterTeamToContestUseCase
	updateTeamRegistration    *appTeam.UpdateTeamRegistrationUseCase
	unregisterTeamFromContest *appTeam.UnregisterTeamFromContestUseCase
	listTeamRegistrations     *appTeam.ListTeamRegistrationsUseCase
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
	registerTeamToContest *appTeam.RegisterTeamToContestUseCase,
	updateTeamRegistration *appTeam.UpdateTeamRegistrationUseCase,
	unregisterTeamFromContest *appTeam.UnregisterTeamFromContestUseCase,
	listTeamRegistrations *appTeam.ListTeamRegistrationsUseCase,
) *Handler {
	return &Handler{
		createTeam:                createTeam,
		listMyTeams:               listMyTeams,
		getTeam:                   getTeam,
		inviteToTeam:              inviteToTeam,
		listMyInvitations:         listMyInvitations,
		acceptInvitation:          acceptInvitation,
		rejectInvitation:          rejectInvitation,
		leaveTeam:                 leaveTeam,
		registerTeamToContest:     registerTeamToContest,
		updateTeamRegistration:    updateTeamRegistration,
		unregisterTeamFromContest: unregisterTeamFromContest,
		listTeamRegistrations:     listTeamRegistrations,
	}
}

func (h *Handler) requireCurrentUser(w http.ResponseWriter, r *http.Request) (*shared.CurrentUser, bool) {
	cu, ok := middleware.GetCurrentUser(r.Context())
	if !ok {
		handler.WriteError(r.Context(), w, apperror.NewUnauthorized(apperror.ErrCodeUnauthorized, "Invalid or missing authentication token"))
		return nil, false
	}
	return &cu, true
}
