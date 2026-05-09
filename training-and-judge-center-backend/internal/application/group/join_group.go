package group

import (
	"context"
	"time"

	"github.com/google/uuid"

	domainGroup "github.com/training-judge-center/backend/internal/domain/group"
	"github.com/training-judge-center/backend/internal/domain/shared"
	appshared "github.com/training-judge-center/backend/internal/application/shared"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type JoinGroupInput struct {
	GroupID     string
	CurrentUser appshared.CurrentUser
}

type JoinGroupOutput struct {
	Member MemberDTO
}

type JoinGroupUseCase struct {
	groupRepo  domainGroup.Repository
	memberRepo domainGroup.MemberRepository
}

func NewJoinGroupUseCase(groupRepo domainGroup.Repository, memberRepo domainGroup.MemberRepository) *JoinGroupUseCase {
	return &JoinGroupUseCase{groupRepo: groupRepo, memberRepo: memberRepo}
}

func (uc *JoinGroupUseCase) Execute(ctx context.Context, input JoinGroupInput) (*JoinGroupOutput, error) {
	if input.GroupID == "" {
		return nil, apperror.NewValidation([]apperror.FieldError{
			{Field: "groupId", Message: "groupId is required"},
		})
	}

	g, err := uc.groupRepo.FindByID(ctx, input.GroupID)
	if err != nil {
		return nil, err
	}

	// Policy check precedes membership check: a non-OPEN group returns 403 regardless of the caller's membership status.
	if g.JoinPolicy() != domainGroup.JoinPolicyOpen {
		return nil, apperror.NewForbidden(ErrCodeInsufficientPermissions, "This group does not allow direct joining")
	}

	userID := shared.RestoreUserID(input.CurrentUser.ID)
	existing, err := uc.memberRepo.FindByGroupAndUser(ctx, input.GroupID, userID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, apperror.NewConflict(ErrCodeAlreadyMember, "You are already a member of this group")
	}

	now := time.Now()
	newMemberID := uuid.New().String()
	member, err := domainGroup.NewGroupMember(newMemberID, input.GroupID, userID, domainGroup.MemberRoleMember, now)
	if err != nil {
		return nil, err
	}

	if err := uc.memberRepo.Save(ctx, member); err != nil {
		return nil, err
	}

	return &JoinGroupOutput{Member: memberToDTO(member)}, nil
}
