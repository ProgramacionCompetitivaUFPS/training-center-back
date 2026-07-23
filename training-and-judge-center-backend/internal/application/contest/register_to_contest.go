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
}

func NewRegisterToContestUseCase(
	repo domainContest.Repository,
	registrationRepo domainContest.RegistrationRepository,
	memberProvider GroupMemberProvider,
	teamSelectionChecker TeamSelectionChecker,
) *RegisterToContestUseCase {
	return &RegisterToContestUseCase{
		repo:                 repo,
		registrationRepo:     registrationRepo,
		memberProvider:       memberProvider,
		teamSelectionChecker: teamSelectionChecker,
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
	if c.Status(now) != domainContest.StatusScheduled {
		return apperror.NewBadRequest(domainContest.ErrCodeContestAlreadyStarted, "registration is only allowed before the contest starts")
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

	exists, err := uc.registrationRepo.ExistsByContestAndUser(ctx, in.ContestID, in.CurrentUser.ID)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}

	inTeam, err := uc.teamSelectionChecker.IsUserSelectedInAnyTeam(ctx, in.ContestID, in.CurrentUser.ID)
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
	return uc.registrationRepo.Save(ctx, reg)
}
