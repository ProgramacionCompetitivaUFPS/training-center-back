package team

import (
	"context"

	"github.com/google/uuid"
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
	teamRepo             domainTeam.Repository
	memberRepo           domainTeam.MemberRepository
	contestChecker       ContestParticipationChecker
	participationCleaner ScheduledParticipationCleaner
	txManager            appShared.TransactionManager
}

func NewRemoveTeamMemberUseCase(
	teamRepo domainTeam.Repository,
	memberRepo domainTeam.MemberRepository,
	contestChecker ContestParticipationChecker,
	participationCleaner ScheduledParticipationCleaner,
	txManager appShared.TransactionManager,
) *RemoveTeamMemberUseCase {
	return &RemoveTeamMemberUseCase{
		teamRepo:             teamRepo,
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
	// A UUID has one identity however it is written (upper or lower case): compare and query by the
	// canonical form, or the same person written in capitals would slip past the self-removal check.
	target, err := uuid.Parse(in.TargetUserID)
	if err != nil {
		return apperror.NewNotFound(domainTeam.ErrCodeNotTeamMember, "That user is not a member of this team")
	}
	if requester, err := uuid.Parse(in.CurrentUser.ID); err == nil && requester == target {
		return apperror.NewBadRequest(domainTeam.ErrCodeCannotRemoveSelf, "You cannot remove yourself; leave the team instead")
	}
	targetID := target.String()

	requesterID := domainShared.RestoreUserID(in.CurrentUser.ID)
	targetUserID := domainShared.RestoreUserID(targetID)

	return uc.txManager.WithTx(ctx, func(txCtx context.Context) error {
		// Everything is decided inside the transaction, after taking a lock on the team. A plain SELECT
		// would not stop two members removing each other at the same moment (both would see the other
		// still on the team, each delete a different row, and the team would end up empty). The second
		// request waits here, then sees what the first one committed.
		if err := uc.teamRepo.LockByID(txCtx, in.TeamID); err != nil {
			return err
		}

		if _, err := uc.memberRepo.FindByTeamAndUser(txCtx, in.TeamID, requesterID); err != nil {
			if ae, ok := err.(*apperror.AppError); ok && ae.Kind == apperror.KindNotFound {
				return apperror.NewForbidden(domainTeam.ErrCodeNotTeamMember, "You are not a member of this team")
			}
			return err
		}
		if _, err := uc.memberRepo.FindByTeamAndUser(txCtx, in.TeamID, targetUserID); err != nil {
			if ae, ok := err.(*apperror.AppError); ok && ae.Kind == apperror.KindNotFound {
				return apperror.NewNotFound(domainTeam.ErrCodeNotTeamMember, "That user is not a member of this team")
			}
			return err
		}

		// Same transaction as the cleanup below, so both read the same NOW(): a contest that starts in
		// between cannot be skipped by the cleanup after the check said it had not started.
		inActive, err := uc.contestChecker.IsUserInActiveContestForTeam(txCtx, targetID, in.TeamID)
		if err != nil {
			return err
		}
		if inActive {
			return apperror.NewConflict(domainTeam.ErrCodeCannotRemoveDuringActiveContest, "That member is selected to participate in an active contest with this team")
		}

		if _, err := uc.participationCleaner.RemoveUserFromScheduledParticipations(txCtx, in.TeamID, targetID); err != nil {
			return err
		}
		return uc.memberRepo.DeleteByTeamAndUser(txCtx, in.TeamID, targetUserID)
	})
}
