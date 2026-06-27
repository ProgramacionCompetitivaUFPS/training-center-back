package contest

import (
	"context"
	"time"

	appshared "github.com/training-judge-center/backend/internal/application/shared"
	domainContest "github.com/training-judge-center/backend/internal/domain/contest"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type GetRegistrationStatusInput struct {
	CurrentUser appshared.CurrentUser
	GroupID     string
	ContestID   string
}

type GetRegistrationStatusOutput struct {
	Registered   bool
	RegisteredAt *time.Time
}

type GetRegistrationStatusUseCase struct {
	repo             domainContest.Repository
	registrationRepo domainContest.RegistrationRepository
}

func NewGetRegistrationStatusUseCase(
	repo domainContest.Repository,
	registrationRepo domainContest.RegistrationRepository,
) *GetRegistrationStatusUseCase {
	return &GetRegistrationStatusUseCase{
		repo:             repo,
		registrationRepo: registrationRepo,
	}
}

func (uc *GetRegistrationStatusUseCase) Execute(ctx context.Context, in GetRegistrationStatusInput) (*GetRegistrationStatusOutput, error) {
	c, err := uc.repo.FindByID(ctx, in.ContestID)
	if err != nil {
		return nil, err
	}
	if c.GroupID().Value() != in.GroupID {
		return nil, apperror.NewNotFound(domainContest.ErrCodeContestNotFound, "contest not found")
	}

	reg, err := uc.registrationRepo.FindByContestAndUser(ctx, in.ContestID, in.CurrentUser.ID)
	if err != nil {
		return nil, err
	}
	if reg == nil {
		return &GetRegistrationStatusOutput{Registered: false}, nil
	}

	t := reg.RegisteredAt()
	return &GetRegistrationStatusOutput{Registered: true, RegisteredAt: &t}, nil
}
