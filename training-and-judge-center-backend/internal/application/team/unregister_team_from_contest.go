package team

import (
	"context"

	appShared "github.com/training-judge-center/backend/internal/application/shared"
	domainContest "github.com/training-judge-center/backend/internal/domain/contest"
	domainShared "github.com/training-judge-center/backend/internal/domain/shared"
	domainTeam "github.com/training-judge-center/backend/internal/domain/team"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type UnregisterTeamFromContestInput struct {
	CurrentUser appShared.CurrentUser
	ContestID   string
	TeamID      string
}

type UnregisterTeamFromContestUseCase struct {
	memberRepo       domainTeam.MemberRepository
	contestProvider  ContestProvider
	teamParticipRepo domainContest.TeamParticipationRepository
}

func NewUnregisterTeamFromContestUseCase(
	memberRepo domainTeam.MemberRepository,
	contestProvider ContestProvider,
	teamParticipRepo domainContest.TeamParticipationRepository,
) *UnregisterTeamFromContestUseCase {
	return &UnregisterTeamFromContestUseCase{
		memberRepo:       memberRepo,
		contestProvider:  contestProvider,
		teamParticipRepo: teamParticipRepo,
	}
}

func (uc *UnregisterTeamFromContestUseCase) Execute(ctx context.Context, in UnregisterTeamFromContestInput) error {
	requesterID := domainShared.RestoreUserID(in.CurrentUser.ID)

	// 1. Requester must be a team member.
	if _, err := uc.memberRepo.FindByTeamAndUser(ctx, in.TeamID, requesterID); err != nil {
		if ae, ok := err.(*apperror.AppError); ok && ae.Kind == apperror.KindNotFound {
			return apperror.NewForbidden(domainTeam.ErrCodeNotTeamMember, "you are not a member of this team")
		}
		return err
	}

	// 2. Contest must be SCHEDULED.
	contest, err := uc.contestProvider.GetContestInfo(ctx, in.ContestID)
	if err != nil {
		return err
	}
	if contest.Status != "SCHEDULED" {
		return apperror.NewConflict(domainContest.ErrCodeCannotUnregisterAfterStart, "cannot unregister after the contest has started")
	}

	return uc.teamParticipRepo.Delete(ctx, in.ContestID, in.TeamID)
}
