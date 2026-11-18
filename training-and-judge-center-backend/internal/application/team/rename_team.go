package team

import (
	"context"

	appShared "github.com/training-judge-center/backend/internal/application/shared"
	domainShared "github.com/training-judge-center/backend/internal/domain/shared"
	domainTeam "github.com/training-judge-center/backend/internal/domain/team"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type RenameTeamInput struct {
	CurrentUser appShared.CurrentUser
	TeamID      string
	Name        string
}

type RenameTeamOutput struct {
	ID   string
	Name string
}

// RenameTeamUseCase lets any member of a team change its name. The name follows the same rules as when
// the team is created: trimmed, within the length limit, and unique ignoring case.
type RenameTeamUseCase struct {
	teamRepo   domainTeam.Repository
	memberRepo domainTeam.MemberRepository
}

func NewRenameTeamUseCase(teamRepo domainTeam.Repository, memberRepo domainTeam.MemberRepository) *RenameTeamUseCase {
	return &RenameTeamUseCase{teamRepo: teamRepo, memberRepo: memberRepo}
}

func (uc *RenameTeamUseCase) Execute(ctx context.Context, in RenameTeamInput) (*RenameTeamOutput, error) {
	name, err := domainTeam.NewTeamName(in.Name)
	if err != nil {
		return nil, err
	}

	team, err := uc.teamRepo.FindByID(ctx, in.TeamID)
	if err != nil {
		return nil, err
	}

	requesterID := domainShared.RestoreUserID(in.CurrentUser.ID)
	if _, err := uc.memberRepo.FindByTeamAndUser(ctx, in.TeamID, requesterID); err != nil {
		if ae, ok := err.(*apperror.AppError); ok && ae.Kind == apperror.KindNotFound {
			return nil, apperror.NewForbidden(domainTeam.ErrCodeNotTeamMember, "You are not a member of this team")
		}
		return nil, err
	}

	team.Rename(name)
	if err := uc.teamRepo.UpdateName(ctx, team); err != nil {
		return nil, err
	}
	return &RenameTeamOutput{ID: team.ID(), Name: team.Name().Value()}, nil
}
