package group

import (
	"context"

	appshared "github.com/training-judge-center/backend/internal/application/shared"
	domainGroup "github.com/training-judge-center/backend/internal/domain/group"
	"github.com/training-judge-center/backend/internal/domain/shared"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type RemoveMemberInput struct {
	GroupID     string
	Nickname    string
	CurrentUser appshared.CurrentUser
}

type RemoveMemberUseCase struct {
	groupRepo        domainGroup.Repository
	memberRepo       domainGroup.MemberRepository
	nicknameResolver NicknameResolver
	contestCleaner   ContestRegistrationCleaner
	txManager        appshared.TransactionManager
}

func NewRemoveMemberUseCase(
	groupRepo domainGroup.Repository,
	memberRepo domainGroup.MemberRepository,
	nicknameResolver NicknameResolver,
	contestCleaner ContestRegistrationCleaner,
	txManager appshared.TransactionManager,
) *RemoveMemberUseCase {
	return &RemoveMemberUseCase{
		groupRepo:        groupRepo,
		memberRepo:       memberRepo,
		nicknameResolver: nicknameResolver,
		contestCleaner:   contestCleaner,
		txManager:        txManager,
	}
}

func (uc *RemoveMemberUseCase) Execute(ctx context.Context, input RemoveMemberInput) error {
	g, err := uc.groupRepo.FindByID(ctx, input.GroupID)
	if err != nil {
		return err
	}

	if g.IsDefault() {
		return apperror.NewBadRequest(domainGroup.ErrCodeCannotRemoveFromGlobalGroup, "cannot remove members from the global group")
	}

	if err := requireLeadOrAdmin(ctx, uc.memberRepo, input.GroupID, input.CurrentUser); err != nil {
		return err
	}

	target, err := uc.nicknameResolver.ResolveByNickname(ctx, input.Nickname)
	if err != nil {
		return err
	}
	if target == nil {
		return apperror.NewNotFound("NICKNAME_NOT_FOUND", "the specified nickname does not exist")
	}

	targetID := shared.RestoreUserID(target.ID)
	member, err := uc.memberRepo.FindByGroupAndUser(ctx, input.GroupID, targetID)
	if err != nil {
		return err
	}
	if member == nil {
		return apperror.NewNotFound(domainGroup.ErrCodeNotAMember, "user is not a member of this group")
	}

	if member.IsLead() {
		leadCount, err := uc.memberRepo.CountLeads(ctx, input.GroupID)
		if err != nil {
			return err
		}
		if leadCount <= 1 {
			return apperror.NewBadRequest(domainGroup.ErrCodeCannotRemoveLastLead, "cannot remove the last lead from the group")
		}
	}

	return uc.txManager.WithTx(ctx, func(txCtx context.Context) error {
		if _, err := uc.contestCleaner.DeleteScheduledByGroupAndUser(txCtx, input.GroupID, targetID.Value()); err != nil {
			return err
		}
		return uc.memberRepo.Delete(txCtx, input.GroupID, targetID)
	})
}
