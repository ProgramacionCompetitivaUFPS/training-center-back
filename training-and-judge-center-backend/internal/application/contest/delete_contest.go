package contest

import (
	"context"
	"time"

	appshared "github.com/training-judge-center/backend/internal/application/shared"
	domainContest "github.com/training-judge-center/backend/internal/domain/contest"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type DeleteContestInput struct {
	CurrentUser appshared.CurrentUser
	GroupID     string
	ContestID   string
	Now         time.Time
}

type DeleteContestUseCase struct {
	contestRepo    domainContest.Repository
	groupProvider  GroupProvider
	memberProvider GroupMemberProvider
	standingsCache StandingsCache
}

func NewDeleteContestUseCase(
	contestRepo domainContest.Repository,
	groupProvider GroupProvider,
	memberProvider GroupMemberProvider,
	standingsCache StandingsCache,
) *DeleteContestUseCase {
	return &DeleteContestUseCase{
		contestRepo:    contestRepo,
		groupProvider:  groupProvider,
		memberProvider: memberProvider,
		standingsCache: standingsCache,
	}
}

func (uc *DeleteContestUseCase) Execute(ctx context.Context, in DeleteContestInput) error {
	contest, err := uc.contestRepo.FindByID(ctx, in.ContestID)
	if err != nil {
		return err
	}
	if contest == nil || contest.GroupID().Value() != in.GroupID {
		return apperror.NewNotFound(domainContest.ErrCodeContestNotFound, "contest not found")
	}

	group, err := uc.groupProvider.FindByID(ctx, in.GroupID)
	if err != nil {
		return err
	}
	if group == nil {
		return apperror.NewNotFound(ErrCodeGroupNotFound, "group not found")
	}

	if contest.Status(in.Now) == domainContest.StatusActive {
		return apperror.NewConflict(domainContest.ErrCodeContestIsActive, "cannot delete an active contest")
	}

	isAdmin := in.CurrentUser.IsAdmin()
	if !isAdmin {
		isLead, err := uc.memberProvider.IsLeadOfGroup(ctx, in.CurrentUser.ID, in.GroupID)
		if err != nil {
			return err
		}
		if !isLead {
			return apperror.NewForbidden(ErrCodeInsufficientPermissions, "only leads and admins can delete contests")
		}
	}

	if err := uc.contestRepo.Delete(ctx, in.ContestID); err != nil {
		return err
	}

	// Best-effort: stale cache is harmless (key won't be served for a deleted contest)
	// but cleaning it up avoids memory waste in Redis.
	_ = uc.standingsCache.Invalidate(ctx, in.ContestID)

	return nil
}
