package team

import (
	"context"
	"time"

	appShared "github.com/training-judge-center/backend/internal/application/shared"
	domainShared "github.com/training-judge-center/backend/internal/domain/shared"
	domainTeam "github.com/training-judge-center/backend/internal/domain/team"
)

type ListMyInvitationsInput struct {
	CurrentUser appShared.CurrentUser
}

type InvitationOutput struct {
	ID        string
	Team      TeamDisplay
	InvitedBy UserDisplay
	CreatedAt time.Time
}

type TeamDisplay struct {
	ID   string
	Name string
}

type ListMyInvitationsOutput struct {
	Invitations []InvitationOutput
}

type ListMyInvitationsUseCase struct {
	invitationRepo domainTeam.InvitationRepository
	teamRepo       domainTeam.Repository
	userProvider   UserProvider
}

func NewListMyInvitationsUseCase(
	invitationRepo domainTeam.InvitationRepository,
	teamRepo domainTeam.Repository,
	userProvider UserProvider,
) *ListMyInvitationsUseCase {
	return &ListMyInvitationsUseCase{
		invitationRepo: invitationRepo,
		teamRepo:       teamRepo,
		userProvider:   userProvider,
	}
}

func (uc *ListMyInvitationsUseCase) Execute(ctx context.Context, in ListMyInvitationsInput) (*ListMyInvitationsOutput, error) {
	inviteeID := domainShared.RestoreUserID(in.CurrentUser.ID)

	invitations, err := uc.invitationRepo.FindByInvitee(ctx, inviteeID)
	if err != nil {
		return nil, err
	}

	if len(invitations) == 0 {
		return &ListMyInvitationsOutput{Invitations: []InvitationOutput{}}, nil
	}

	teamIDSet := make(map[string]struct{}, len(invitations))
	inviterIDSet := make(map[string]struct{}, len(invitations))
	for _, inv := range invitations {
		teamIDSet[inv.TeamID()] = struct{}{}
		inviterIDSet[inv.InvitedBy().Value()] = struct{}{}
	}

	teamIDs := make([]string, 0, len(teamIDSet))
	for id := range teamIDSet {
		teamIDs = append(teamIDs, id)
	}
	inviterIDs := make([]string, 0, len(inviterIDSet))
	for id := range inviterIDSet {
		inviterIDs = append(inviterIDs, id)
	}

	teams, err := uc.teamRepo.FindByIDs(ctx, teamIDs)
	if err != nil {
		return nil, err
	}
	teamsByID := make(map[string]*domainTeam.Team, len(teams))
	for _, t := range teams {
		teamsByID[t.ID()] = t
	}

	displays, err := uc.userProvider.GetDisplays(ctx, inviterIDs)
	if err != nil {
		return nil, err
	}

	nickname := func(id string) string {
		if d := displays[id]; d != nil {
			return d.Nickname
		}
		return ""
	}

	outputs := make([]InvitationOutput, 0, len(invitations))
	for _, inv := range invitations {
		t := teamsByID[inv.TeamID()]
		teamName := ""
		if t != nil {
			teamName = t.Name().Value()
		}
		outputs = append(outputs, InvitationOutput{
			ID:        inv.ID(),
			Team:      TeamDisplay{ID: inv.TeamID(), Name: teamName},
			InvitedBy: UserDisplay{ID: inv.InvitedBy().Value(), Nickname: nickname(inv.InvitedBy().Value())},
			CreatedAt: inv.CreatedAt(),
		})
	}

	return &ListMyInvitationsOutput{Invitations: outputs}, nil
}
