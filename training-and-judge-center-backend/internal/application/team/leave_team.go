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
	memberRepo          domainTeam.MemberRepository
	contestChecker      ContestParticipationChecker
	participationCleaner ScheduledParticipationCleaner
	txManager           appShared.TransactionManager
}

func NewLeaveTeamUseCase(
	memberRepo domainTeam.MemberRepository,
	contestChecker ContestParticipationChecker,
	participationCleaner ScheduledParticipationCleaner,
	txManager appShared.TransactionManager,
) *LeaveTeamUseCase {
	return &LeaveTeamUseCase{
		memberRepo:          memberRepo,
		contestChecker:      contestChecker,
		participationCleaner: participationCleaner,
		txManager:           txManager,
	}
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

	return uc.txManager.WithTx(ctx, func(txCtx context.Context) error {
		if _, err := uc.participationCleaner.RemoveUserFromScheduledParticipations(txCtx, in.TeamID, in.CurrentUser.ID); err != nil {
			return err
		}
		return uc.memberRepo.DeleteByTeamAndUser(txCtx, in.TeamID, userID)
	})
}
