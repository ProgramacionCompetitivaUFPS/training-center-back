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

type RegisterToContestUseCase struct {
	repo                 domainContest.Repository
	registrationRepo     domainContest.RegistrationRepository
	memberProvider       GroupMemberProvider
	teamSelectionChecker TeamSelectionChecker
	registrationLocker   RegistrationLocker
	txManager            appshared.TransactionManager
}

func NewRegisterToContestUseCase(
	repo domainContest.Repository,
	registrationRepo domainContest.RegistrationRepository,
	memberProvider GroupMemberProvider,
	teamSelectionChecker TeamSelectionChecker,
	registrationLocker RegistrationLocker,
	txManager appshared.TransactionManager,
) *RegisterToContestUseCase {
	return &RegisterToContestUseCase{
		repo:                 repo,
		registrationRepo:     registrationRepo,
		memberProvider:       memberProvider,
		teamSelectionChecker: teamSelectionChecker,
		registrationLocker:   registrationLocker,
		txManager:            txManager,
	}
}

func (uc *RegisterToContestUseCase) Execute(ctx context.Context, in RegisterToContestInput) error {
	c, err := uc.repo.FindByID(ctx, in.ContestID)
	if err != nil {
		return err
	}
	if c.GroupID().Value() != in.GroupID {
		return apperror.NewNotFound(domainContest.ErrCodeContestNotFound, "contest not found")
	}

	now := time.Now()
	if err := c.CheckRegistrationOpen(now); err != nil {
		return err
	}

	if in.CurrentUser.IsAdmin() {
		return apperror.NewForbidden(ErrCodeAdminsCannotRegister, "admins cannot register to contests")
	}

	role, err := uc.memberProvider.GetMemberRole(ctx, in.CurrentUser.ID, in.GroupID)
	if err != nil {
		return err
	}
	if role != nil && *role == "LEAD" {
		return apperror.NewForbidden(ErrCodeLeadsCannotRegister, "leads cannot register to contests")
	}
	if role == nil {
		return apperror.NewForbidden(ErrCodeNotGroupMember, "only group members can register to this contest")
	}

	// The team-registration use cases take the same lock, so a user cannot end up both
	// registered individually and selected in a team.
	return uc.txManager.WithTx(ctx, func(txCtx context.Context) error {
		if err := uc.registrationLocker.Lock(txCtx, in.ContestID); err != nil {
			return err
		}

		exists, err := uc.registrationRepo.ExistsByContestAndUser(txCtx, in.ContestID, in.CurrentUser.ID)
		if err != nil {
			return err
		}
		if exists {
			return nil
		}

		inTeam, err := uc.teamSelectionChecker.IsUserSelectedInAnyTeam(txCtx, in.ContestID, in.CurrentUser.ID)
		if err != nil {
			return err
		}
		if inTeam {
			return apperror.NewConflict(ErrCodeAlreadyInTeam, "you are already registered to this contest as a selected team member")
		}

		reg, err := domainContest.NewContestRegistration(uuid.New().String(), in.ContestID, in.CurrentUser.ID, now)
		if err != nil {
			return err
		}
		return uc.registrationRepo.Save(txCtx, reg)
	})
}
