package team

import (
	"context"

	appShared "github.com/training-judge-center/backend/internal/application/shared"
	domainShared "github.com/training-judge-center/backend/internal/domain/shared"
	domainTeam "github.com/training-judge-center/backend/internal/domain/team"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type RemoveTeamMemberInput struct {
	CurrentUser  appShared.CurrentUser
	TeamID       string
	TargetUserID string
}

// RemoveTeamMemberUseCase lets any member of a team take another member off it. It follows the same
// rules as that member leaving: it is refused while they are selected in a running contest, and the
// registrations of contests that have not started lose them (or are cancelled if that leaves the team
// below the contest's minimum).
type RemoveTeamMemberUseCase struct {
	memberRepo           domainTeam.MemberRepository
	contestChecker       ContestParticipationChecker
	participationCleaner ScheduledParticipationCleaner
	txManager            appShared.TransactionManager
}

func NewRemoveTeamMemberUseCase(
	memberRepo domainTeam.MemberRepository,
	contestChecker ContestParticipationChecker,
	participationCleaner ScheduledParticipationCleaner,
	txManager appShared.TransactionManager,
) *RemoveTeamMemberUseCase {
	return &RemoveTeamMemberUseCase{
		memberRepo:           memberRepo,
		contestChecker:       contestChecker,
		participationCleaner: participationCleaner,
		txManager:            txManager,
	}
}

func (uc *RemoveTeamMemberUseCase) Execute(ctx context.Context, in RemoveTeamMemberInput) error {
	if in.TargetUserID == in.CurrentUser.ID {
		return apperror.NewBadRequest(domainTeam.ErrCodeCannotRemoveSelf, "You cannot remove yourself; leave the team instead")
	}

	requesterID := domainShared.RestoreUserID(in.CurrentUser.ID)
	if _, err := uc.memberRepo.FindByTeamAndUser(ctx, in.TeamID, requesterID); err != nil {
		if ae, ok := err.(*apperror.AppError); ok && ae.Kind == apperror.KindNotFound {
			return apperror.NewForbidden(domainTeam.ErrCodeNotTeamMember, "You are not a member of this team")
		}
		return err
	}

	targetID := domainShared.RestoreUserID(in.TargetUserID)
	if _, err := uc.memberRepo.FindByTeamAndUser(ctx, in.TeamID, targetID); err != nil {
		if ae, ok := err.(*apperror.AppError); ok && ae.Kind == apperror.KindNotFound {
			return apperror.NewNotFound(domainTeam.ErrCodeNotTeamMember, "That user is not a member of this team")
		}
		return err
	}

	inActive, err := uc.contestChecker.IsUserInActiveContestForTeam(ctx, in.TargetUserID, in.TeamID)
	if err != nil {
		return err
	}
	if inActive {
		return apperror.NewConflict(domainTeam.ErrCodeCannotRemoveDuringActiveContest, "That member is selected to participate in an active contest with this team")
	}

	return uc.txManager.WithTx(ctx, func(txCtx context.Context) error {
		// Two members removing each other at the same moment must not empty the team: check again,
		// inside the transaction, that the requester is still on it.
		if _, err := uc.memberRepo.FindByTeamAndUser(txCtx, in.TeamID, requesterID); err != nil {
			if ae, ok := err.(*apperror.AppError); ok && ae.Kind == apperror.KindNotFound {
				return apperror.NewForbidden(domainTeam.ErrCodeNotTeamMember, "You are not a member of this team")
			}
			return err
		}
		if _, err := uc.participationCleaner.RemoveUserFromScheduledParticipations(txCtx, in.TeamID, in.TargetUserID); err != nil {
			return err
		}
		return uc.memberRepo.DeleteByTeamAndUser(txCtx, in.TeamID, targetID)
	})
}
