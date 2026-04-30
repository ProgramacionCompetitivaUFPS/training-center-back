package group

import (
	"context"

	"github.com/google/uuid"
	domainGroup "github.com/training-judge-center/backend/internal/domain/group"
	"github.com/training-judge-center/backend/internal/domain/shared"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type AddMemberInput struct {
	GroupID     string
	Nickname    string
	Role        string
	CurrentUser shared.CurrentUser
}

type AddMemberResult struct {
	Member   *domainGroup.GroupMember
	Nickname string
}

type AddMemberUseCase struct {
	groupRepo  domainGroup.Repository
	memberRepo domainGroup.MemberRepository
	resolver   NicknameResolver
	tx         TransactionManager
}

func NewAddMemberUseCase(
	groupRepo domainGroup.Repository,
	memberRepo domainGroup.MemberRepository,
	resolver NicknameResolver,
	tx TransactionManager,
) *AddMemberUseCase {
	return &AddMemberUseCase{groupRepo: groupRepo, memberRepo: memberRepo, resolver: resolver, tx: tx}
}

func (uc *AddMemberUseCase) Execute(ctx context.Context, input AddMemberInput) (*AddMemberResult, error) {
	targetInfo, err := uc.resolver.ResolveByNickname(ctx, input.Nickname)
	if err != nil {
		return nil, err
	}
	if targetInfo == nil {
		return nil, apperror.NewNotFound(domainGroup.ErrCodeNicknameNotFound, "The specified nickname does not exist")
	}

	if _, err := uc.groupRepo.FindByID(ctx, input.GroupID); err != nil {
		return nil, err
	}

	if !input.CurrentUser.IsAdmin() {
		callerMembership, err := uc.memberRepo.FindByGroupAndUser(ctx, input.GroupID, shared.RestoreUserID(input.CurrentUser.ID))
		if err != nil {
			return nil, err
		}
		if callerMembership == nil || !callerMembership.IsLead() {
			return nil, apperror.NewForbidden(domainGroup.ErrCodeInsufficientPermissions, "Only leads can add members to the group")
		}
	}

	targetRole, err := domainGroup.NewMemberRole(input.Role)
	if err != nil {
		return nil, err
	}

	if targetRole == domainGroup.MemberRoleLead &&
		targetInfo.Role != shared.RoleCoach && targetInfo.Role != shared.RoleAdmin {
		return nil, apperror.NewBadRequest(domainGroup.ErrCodeInvalidLeadAssignment, "Only Coaches and Admins can be assigned as leads")
	}

	actorID := shared.RestoreUserID(input.CurrentUser.ID)
	targetUserID := shared.RestoreUserID(targetInfo.ID)

	var result *AddMemberResult
	txErr := uc.tx.WithTx(ctx, func(txCtx context.Context) error {
		existing, err := uc.memberRepo.FindByGroupAndUser(txCtx, input.GroupID, targetUserID)
		if err != nil {
			return err
		}
		if existing != nil {
			return apperror.NewConflict(domainGroup.ErrCodeAlreadyMember, "User is already a member of this group")
		}

		member, err := domainGroup.NewGroupMember(
			uuid.New().String(), input.GroupID,
			targetUserID, targetRole,
			&actorID, domainGroup.JoinMethodDirectAdd, nil,
		)
		if err != nil {
			return err
		}
		if err := uc.memberRepo.Save(txCtx, member); err != nil {
			return err
		}

		result = &AddMemberResult{Member: member, Nickname: input.Nickname}
		return nil
	})
	if txErr != nil {
		return nil, txErr
	}
	return result, nil
}
