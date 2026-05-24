package contest

import (
	"context"
	"time"

	"github.com/google/uuid"
	appshared "github.com/training-judge-center/backend/internal/application/shared"
	domainContest "github.com/training-judge-center/backend/internal/domain/contest"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type RegisterToContestInput struct {
	CurrentUser appshared.CurrentUser
	GroupID     string
	ContestID   string
}

type RegisterToContestOutput struct {
	RegisteredAt time.Time
}

type RegisterToContestUseCase struct {
	repo             domainContest.Repository
	registrationRepo domainContest.RegistrationRepository
	groupProvider    GroupProvider
	memberProvider   GroupMemberProvider
}

func NewRegisterToContestUseCase(
	repo domainContest.Repository,
	registrationRepo domainContest.RegistrationRepository,
	groupProvider GroupProvider,
	memberProvider GroupMemberProvider,
) *RegisterToContestUseCase {
	return &RegisterToContestUseCase{
		repo:             repo,
		registrationRepo: registrationRepo,
		groupProvider:    groupProvider,
		memberProvider:   memberProvider,
	}
}

func (uc *RegisterToContestUseCase) Execute(ctx context.Context, in RegisterToContestInput) (*RegisterToContestOutput, error) {
	c, err := uc.repo.FindByID(ctx, in.ContestID)
	if err != nil {
		return nil, err
	}
	if c == nil {
		return nil, apperror.NewNotFound(domainContest.ErrCodeContestNotFound, "contest not found")
	}

	if c.GroupID().Value() != in.GroupID {
		return nil, apperror.NewNotFound(domainContest.ErrCodeContestNotFound, "contest not found")
	}

	now := time.Now()
	if c.Status(now) == domainContest.StatusFinished {
		return nil, apperror.NewConflict(domainContest.ErrCodeRegistrationClosed, "registration is closed for finished contests")
	}

	isAdmin := in.CurrentUser.IsAdmin()
	if !isAdmin {
		isMember, err := uc.memberProvider.IsMemberOfGroup(ctx, in.CurrentUser.ID, in.GroupID)
		if err != nil {
			return nil, err
		}
		if !isMember {
			return nil, apperror.NewForbidden(ErrCodeNotGroupMember, "only group members can register to this contest")
		}
	}

	exists, err := uc.registrationRepo.ExistsByContestAndUser(ctx, in.ContestID, in.CurrentUser.ID)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, apperror.NewConflict(domainContest.ErrCodeAlreadyRegistered, "you are already registered for this contest")
	}

	reg := domainContest.NewContestRegistration(uuid.New().String(), in.ContestID, in.CurrentUser.ID, now)
	if err := uc.registrationRepo.Save(ctx, reg); err != nil {
		return nil, err
	}

	return &RegisterToContestOutput{RegisteredAt: reg.RegisteredAt()}, nil
}
