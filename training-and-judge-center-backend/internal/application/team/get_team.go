package team

import (
	"context"
	"time"

	appshared "github.com/training-judge-center/backend/internal/application/shared"
	domainTeam "github.com/training-judge-center/backend/internal/domain/team"
)

type GetTeamInput struct {
	CurrentUser appshared.CurrentUser
	TeamID      string
}

type TeamMemberOutput struct {
	UserID   string
	Nickname string
	JoinedAt time.Time
}

type TeamCreatorOutput struct {
	UserID   string
	Nickname string
}

type PendingInvitationOutput struct {
	ID        string
	Invitee   UserDisplay
	InvitedBy UserDisplay
	InvitedAt time.Time
}

type GetTeamOutput struct {
	ID                 string
	Name               string
	CreatedBy          TeamCreatorOutput
	CreatedAt          time.Time
	Members            []TeamMemberOutput
	PendingInvitations []PendingInvitationOutput
}

type GetTeamUseCase struct {
	teamRepo       domainTeam.Repository
	memberRepo     domainTeam.MemberRepository
	invitationRepo domainTeam.InvitationRepository
	userProvider   UserProvider
}

func NewGetTeamUseCase(
	teamRepo domainTeam.Repository,
	memberRepo domainTeam.MemberRepository,
	invitationRepo domainTeam.InvitationRepository,
	userProvider UserProvider,
) *GetTeamUseCase {
	return &GetTeamUseCase{
		teamRepo:       teamRepo,
		memberRepo:     memberRepo,
		invitationRepo: invitationRepo,
		userProvider:   userProvider,
	}
}

func (uc *GetTeamUseCase) Execute(ctx context.Context, in GetTeamInput) (*GetTeamOutput, error) {
	team, err := uc.teamRepo.FindByID(ctx, in.TeamID)
	if err != nil {
		return nil, err
	}

	members, err := uc.memberRepo.FindByTeam(ctx, team.ID())
	if err != nil {
		return nil, err
	}

	isMember := in.CurrentUser.ID == team.CreatedBy().Value()
	for _, m := range members {
		if m.UserID().Value() == in.CurrentUser.ID {
			isMember = true
			break
		}
	}

	seen := make(map[string]struct{}, len(members)+1)
	userIDs := make([]string, 0, len(members)+1)
	addID := func(id string) {
		if _, ok := seen[id]; !ok {
			seen[id] = struct{}{}
			userIDs = append(userIDs, id)
		}
	}
	addID(team.CreatedBy().Value())
	for _, m := range members {
		addID(m.UserID().Value())
	}

	var invitations []*domainTeam.TeamInvitation
	if isMember {
		invitations, err = uc.invitationRepo.FindByTeam(ctx, team.ID())
		if err != nil {
			return nil, err
		}
		for _, inv := range invitations {
			addID(inv.InviteeID().Value())
			addID(inv.InvitedBy().Value())
		}
	}

	displays, err := uc.userProvider.GetDisplays(ctx, userIDs)
	if err != nil {
		return nil, err
	}

	display := func(uid string) UserDisplay {
		if d := displays[uid]; d != nil {
			return *d
		}
		return UserDisplay{ID: uid}
	}

	memberOutputs := make([]TeamMemberOutput, len(members))
	for i, m := range members {
		memberOutputs[i] = TeamMemberOutput{
			UserID:   m.UserID().Value(),
			Nickname: display(m.UserID().Value()).Nickname,
			JoinedAt: m.JoinedAt(),
		}
	}

	invOutputs := make([]PendingInvitationOutput, 0, len(invitations))
	for _, inv := range invitations {
		invOutputs = append(invOutputs, PendingInvitationOutput{
			ID:        inv.ID(),
			Invitee:   display(inv.InviteeID().Value()),
			InvitedBy: display(inv.InvitedBy().Value()),
			InvitedAt: inv.CreatedAt(),
		})
	}

	creatorID := team.CreatedBy().Value()
	return &GetTeamOutput{
		ID:   team.ID(),
		Name: team.Name().Value(),
		CreatedBy: TeamCreatorOutput{
			UserID:   creatorID,
			Nickname: display(creatorID).Nickname,
		},
		CreatedAt:          team.CreatedAt(),
		Members:            memberOutputs,
		PendingInvitations: invOutputs,
	}, nil
}
