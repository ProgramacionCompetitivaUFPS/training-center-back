package team

import (
	"context"

	appShared "github.com/training-judge-center/backend/internal/application/shared"
	domainShared "github.com/training-judge-center/backend/internal/domain/shared"
	domainTeam "github.com/training-judge-center/backend/internal/domain/team"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type LeaveTeamInput struct {
	CurrentUser appShared.CurrentUser
	TeamID      string
}

type LeaveTeamUseCase struct {
	memberRepo       domainTeam.MemberRepository
	contestChecker   ContestParticipationChecker
}

func NewLeaveTeamUseCase(
	memberRepo domainTeam.MemberRepository,
	contestChecker ContestParticipationChecker,
) *LeaveTeamUseCase {
	return &LeaveTeamUseCase{memberRepo: memberRepo, contestChecker: contestChecker}
}

func (uc *LeaveTeamUseCase) Execute(ctx context.Context, in LeaveTeamInput) error {
	userID := domainShared.RestoreUserID(in.CurrentUser.ID)

	if _, err := uc.memberRepo.FindByTeamAndUser(ctx, in.TeamID, userID); err != nil {
		if ae, ok := err.(*apperror.AppError); ok && ae.Kind == apperror.KindNotFound {
			return apperror.NewNotFound(domainTeam.ErrCodeNotTeamMember, "You are not a member of this team")
		}
		return err
	}

	inActive, err := uc.contestChecker.IsUserInActiveContestForTeam(ctx, in.CurrentUser.ID, in.TeamID)
	if err != nil {
		return err
	}
	if inActive {
		return apperror.NewConflict(domainTeam.ErrCodeCannotLeaveDuringActiveContest, "You are selected to participate in an active contest with this team")
	}

	return uc.memberRepo.DeleteByTeamAndUser(ctx, in.TeamID, userID)
}
