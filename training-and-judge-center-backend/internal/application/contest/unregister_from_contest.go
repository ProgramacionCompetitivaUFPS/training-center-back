package contest

import (
	"context"
	"time"

	appshared "github.com/training-judge-center/backend/internal/application/shared"
	domainContest "github.com/training-judge-center/backend/internal/domain/contest"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type UnregisterFromContestInput struct {
	CurrentUser appshared.CurrentUser
	GroupID     string
	ContestID   string
}

type UnregisterFromContestUseCase struct {
	repo             domainContest.Repository
	registrationRepo domainContest.RegistrationRepository
	memberProvider   GroupMemberProvider
}

func NewUnregisterFromContestUseCase(
	repo domainContest.Repository,
	registrationRepo domainContest.RegistrationRepository,
	memberProvider GroupMemberProvider,
) *UnregisterFromContestUseCase {
	return &UnregisterFromContestUseCase{
		repo:             repo,
		registrationRepo: registrationRepo,
		memberProvider:   memberProvider,
	}
}

func (uc *UnregisterFromContestUseCase) Execute(ctx context.Context, in UnregisterFromContestInput) error {
	c, err := uc.repo.FindByID(ctx, in.ContestID)
	if err != nil {
		return err
	}
	if c == nil {
		return apperror.NewNotFound(domainContest.ErrCodeContestNotFound, "contest not found")
	}

	if c.GroupID().Value() != in.GroupID {
		return apperror.NewNotFound(domainContest.ErrCodeContestNotFound, "contest not found")
	}

	if c.Status(time.Now()) != domainContest.StatusScheduled {
		return apperror.NewBadRequest(domainContest.ErrCodeCannotUnregisterAfterStart, "unregistration is only allowed before the contest starts")
	}

	isMember, err := uc.memberProvider.IsMemberOfGroup(ctx, in.CurrentUser.ID, in.GroupID)
	if err != nil {
		return err
	}
	if !isMember {
		return apperror.NewForbidden(ErrCodeNotGroupMember, "only group members can unregister from this contest")
	}

	return uc.registrationRepo.Delete(ctx, in.ContestID, in.CurrentUser.ID)
}
