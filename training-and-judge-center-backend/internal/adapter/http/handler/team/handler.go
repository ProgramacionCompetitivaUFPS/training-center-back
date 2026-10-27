package team

import (
	appTeam "github.com/training-judge-center/backend/internal/application/team"
)

type Handler struct {
	createTeam                     *appTeam.CreateTeamUseCase
	listMyTeams                    *appTeam.ListMyTeamsUseCase
	getTeam                        *appTeam.GetTeamUseCase
	inviteToTeam                   *appTeam.InviteToTeamUseCase
	listMyInvitations              *appTeam.ListMyInvitationsUseCase
	acceptInvitation               *appTeam.AcceptInvitationUseCase
	rejectInvitation               *appTeam.RejectInvitationUseCase
	leaveTeam                      *appTeam.LeaveTeamUseCase
	getLeaveImpact                 *appTeam.GetLeaveImpactUseCase
	registerTeamToContest          *appTeam.RegisterTeamToContestUseCase
	updateTeamRegistration         *appTeam.UpdateTeamRegistrationUseCase
	unregisterTeamFromContest      *appTeam.UnregisterTeamFromContestUseCase
	listTeamRegistrations          *appTeam.ListTeamRegistrationsUseCase
	getTeamRegistrationEligibility *appTeam.GetTeamRegistrationEligibilityUseCase
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
	getLeaveImpact *appTeam.GetLeaveImpactUseCase,
	registerTeamToContest *appTeam.RegisterTeamToContestUseCase,
	updateTeamRegistration *appTeam.UpdateTeamRegistrationUseCase,
	unregisterTeamFromContest *appTeam.UnregisterTeamFromContestUseCase,
	listTeamRegistrations *appTeam.ListTeamRegistrationsUseCase,
	getTeamRegistrationEligibility *appTeam.GetTeamRegistrationEligibilityUseCase,
) *Handler {
	return &Handler{
		createTeam:                     createTeam,
		listMyTeams:                    listMyTeams,
		getTeam:                        getTeam,
		inviteToTeam:                   inviteToTeam,
		listMyInvitations:              listMyInvitations,
		acceptInvitation:               acceptInvitation,
		rejectInvitation:               rejectInvitation,
		leaveTeam:                      leaveTeam,
		getLeaveImpact:                 getLeaveImpact,
		registerTeamToContest:          registerTeamToContest,
		updateTeamRegistration:         updateTeamRegistration,
		unregisterTeamFromContest:      unregisterTeamFromContest,
		listTeamRegistrations:          listTeamRegistrations,
		getTeamRegistrationEligibility: getTeamRegistrationEligibility,
	}
}
