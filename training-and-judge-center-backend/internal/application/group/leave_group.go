package group

import (
	"context"

	appshared "github.com/training-judge-center/backend/internal/application/shared"
	domainGroup "github.com/training-judge-center/backend/internal/domain/group"
	"github.com/training-judge-center/backend/internal/domain/shared"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type LeaveGroupInput struct {
	GroupID     string
	CurrentUser appshared.CurrentUser
}

type LeaveGroupUseCase struct {
	groupRepo      domainGroup.Repository
	memberRepo     domainGroup.MemberRepository
	contestCleaner ContestRegistrationCleaner
	txManager      appshared.TransactionManager
}

func NewLeaveGroupUseCase(
	groupRepo domainGroup.Repository,
	memberRepo domainGroup.MemberRepository,
	contestCleaner ContestRegistrationCleaner,
	txManager appshared.TransactionManager,
) *LeaveGroupUseCase {
	return &LeaveGroupUseCase{
		groupRepo:      groupRepo,
		memberRepo:     memberRepo,
		contestCleaner: contestCleaner,
		txManager:      txManager,
	}
}

func (uc *LeaveGroupUseCase) Execute(ctx context.Context, input LeaveGroupInput) error {
	g, err := uc.groupRepo.FindByID(ctx, input.GroupID)
	if err != nil {
		return err
	}

	if g.IsDefault() {
		return apperror.NewBadRequest(domainGroup.ErrCodeCannotLeaveGlobalGroup, "cannot leave the global group")
	}

	callerID := shared.RestoreUserID(input.CurrentUser.ID)
	member, err := uc.memberRepo.FindByGroupAndUser(ctx, input.GroupID, callerID)
	if err != nil {
		return err
	}
	if member == nil {
		return apperror.NewNotFound(domainGroup.ErrCodeNotAMember, "you are not a member of this group")
	}

	if member.IsLead() {
		leadCount, err := uc.memberRepo.CountLeads(ctx, input.GroupID)
		if err != nil {
			return err
		}
		if leadCount <= 1 {
			return apperror.NewBadRequest(domainGroup.ErrCodeCannotLeaveAsLastLead, "cannot leave as the last lead of the group")
		}
	}

	return uc.txManager.WithTx(ctx, func(txCtx context.Context) error {
		if _, err := uc.contestCleaner.DeleteScheduledByGroupAndUser(txCtx, input.GroupID, callerID.Value()); err != nil {
			return err
		}
		return uc.memberRepo.Delete(txCtx, input.GroupID, callerID)
	})
}
