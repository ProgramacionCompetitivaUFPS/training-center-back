package contest

import (
	"context"
	"time"

	appshared "github.com/training-judge-center/backend/internal/application/shared"
	domainContest "github.com/training-judge-center/backend/internal/domain/contest"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type RevealContestResultsInput struct {
	CurrentUser appshared.CurrentUser
	GroupID     string
	ContestID   string
	Now         time.Time
}

type RevealContestResultsUseCase struct {
	contestRepo    domainContest.Repository
	memberProvider GroupMemberProvider
	txManager      appshared.TransactionManager
}

func NewRevealContestResultsUseCase(
	contestRepo domainContest.Repository,
	memberProvider GroupMemberProvider,
	txManager appshared.TransactionManager,
) *RevealContestResultsUseCase {
	return &RevealContestResultsUseCase{
		contestRepo:    contestRepo,
		memberProvider: memberProvider,
		txManager:      txManager,
	}
}

func (uc *RevealContestResultsUseCase) Execute(ctx context.Context, in RevealContestResultsInput) error {
	contest, err := uc.contestRepo.FindByID(ctx, in.ContestID)
	if err != nil {
		return err
	}
	if contest.GroupID().Value() != in.GroupID {
		return apperror.NewNotFound(domainContest.ErrCodeContestNotFound, "contest not found")
	}

	if !in.CurrentUser.IsAdmin() {
		role, err := uc.memberProvider.GetMemberRole(ctx, in.CurrentUser.ID, in.GroupID)
		if err != nil {
			return err
		}
		if role == nil || *role != "LEAD" {
			return apperror.NewForbidden(ErrCodeInsufficientPermissions, "only leads and admins can reveal the results of a contest")
		}
	}

	alreadyRevealed := contest.ResultsRevealedAt() != nil
	if err := contest.RevealResults(in.Now); err != nil {
		return err
	}
	// Already open, by hand or by its automatic time: there is nothing new to save.
	if alreadyRevealed || contest.ResultsRevealedAt() == nil {
		return nil
	}

	// The standings cache holds raw data and the freeze is applied when reading, so it needs no invalidation.
	return uc.txManager.WithTx(ctx, func(txCtx context.Context) error {
		return uc.contestRepo.Update(txCtx, contest)
	})
}
