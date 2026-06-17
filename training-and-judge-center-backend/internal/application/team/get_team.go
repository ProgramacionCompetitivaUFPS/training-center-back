package team

import (
	"context"
	"time"

	domainTeam "github.com/training-judge-center/backend/internal/domain/team"
)

type GetTeamInput struct {
	TeamID string
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

type GetTeamOutput struct {
	ID        string
	Name      string
	CreatedBy TeamCreatorOutput
	CreatedAt time.Time
	Members   []TeamMemberOutput
}

type GetTeamUseCase struct {
	teamRepo     domainTeam.Repository
	memberRepo   domainTeam.MemberRepository
	userProvider UserProvider
}

func NewGetTeamUseCase(teamRepo domainTeam.Repository, memberRepo domainTeam.MemberRepository, userProvider UserProvider) *GetTeamUseCase {
	return &GetTeamUseCase{teamRepo: teamRepo, memberRepo: memberRepo, userProvider: userProvider}
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

	displays, err := uc.userProvider.GetDisplays(ctx, userIDs)
	if err != nil {
		return nil, err
	}

	nickname := func(uid string) string {
		if d := displays[uid]; d != nil {
			return d.Nickname
		}
		return ""
	}

	memberOutputs := make([]TeamMemberOutput, len(members))
	for i, m := range members {
		memberOutputs[i] = TeamMemberOutput{
			UserID:   m.UserID().Value(),
			Nickname: nickname(m.UserID().Value()),
			JoinedAt: m.JoinedAt(),
		}
	}

	creatorID := team.CreatedBy().Value()
	return &GetTeamOutput{
		ID:   team.ID(),
		Name: team.Name().Value(),
		CreatedBy: TeamCreatorOutput{
			UserID:   creatorID,
			Nickname: nickname(creatorID),
		},
		CreatedAt: team.CreatedAt(),
		Members:   memberOutputs,
	}, nil
}

