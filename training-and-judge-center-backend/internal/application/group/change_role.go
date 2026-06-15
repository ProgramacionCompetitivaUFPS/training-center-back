package group

import (
	"context"
	"time"

	appshared "github.com/training-judge-center/backend/internal/application/shared"
	domainGroup "github.com/training-judge-center/backend/internal/domain/group"
	"github.com/training-judge-center/backend/internal/domain/shared"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type ChangeRoleInput struct {
	GroupID     string
	Nickname    string
	Role        string
	CurrentUser appshared.CurrentUser
}

type ChangeRoleOutput struct {
	GroupID       string
	UserID        string
	Nickname      string
	Role          string
	JoinedAt      time.Time
	RoleChangedAt time.Time
}

type ChangeRoleUseCase struct {
	groupRepo        domainGroup.Repository
	memberRepo       domainGroup.MemberRepository
	nicknameResolver NicknameResolver
}

func NewChangeRoleUseCase(
	groupRepo domainGroup.Repository,
	memberRepo domainGroup.MemberRepository,
	nicknameResolver NicknameResolver,
) *ChangeRoleUseCase {
	return &ChangeRoleUseCase{
		groupRepo:        groupRepo,
		memberRepo:       memberRepo,
		nicknameResolver: nicknameResolver,
	}
}

func (uc *ChangeRoleUseCase) Execute(ctx context.Context, input ChangeRoleInput) (*ChangeRoleOutput, error) {
	if input.Role == "" {
		return nil, apperror.NewValidation([]apperror.FieldError{
			{Field: "role", Message: "role is required"},
		})
	}

	newRole, err := domainGroup.NewMemberRole(input.Role)
	if err != nil {
		return nil, err
	}

	if err := requireLeadOrAdmin(ctx, uc.memberRepo, input.GroupID, input.CurrentUser); err != nil {
		return nil, err
	}

	target, err := uc.nicknameResolver.ResolveByNickname(ctx, input.Nickname)
	if err != nil {
		return nil, err
	}
	if target == nil {
		return nil, apperror.NewNotFound("NICKNAME_NOT_FOUND", "the specified nickname does not exist")
	}

	if newRole == domainGroup.MemberRoleLead && target.SystemRole == shared.RoleContestant.String() {
		return nil, apperror.NewBadRequest(domainGroup.ErrCodeInvalidLeadAssignment, "only Coaches and Admins can be assigned as leads")
	}

	targetID := shared.RestoreUserID(target.ID)
	member, err := uc.memberRepo.FindByGroupAndUser(ctx, input.GroupID, targetID)
	if err != nil {
		return nil, err
	}
	if member == nil {
		return nil, apperror.NewNotFound(domainGroup.ErrCodeNotAMember, "user is not a member of this group")
	}

	if newRole == domainGroup.MemberRoleMember && member.IsLead() {
		leadCount, err := uc.memberRepo.CountLeads(ctx, input.GroupID)
		if err != nil {
			return nil, err
		}
		if leadCount <= 1 {
			return nil, apperror.NewBadRequest(domainGroup.ErrCodeCannotRemoveLastLead, "cannot demote the last lead of the group")
		}
	}

	if newRole == domainGroup.MemberRoleLead {
		if err := member.Promote(); err != nil {
			return nil, err
		}
	} else {
		if err := member.Demote(); err != nil {
			return nil, err
		}
	}

	if err := uc.memberRepo.Update(ctx, member); err != nil {
		return nil, err
	}

	now := time.Now()
	return &ChangeRoleOutput{
		GroupID:       member.GroupID(),
		UserID:        member.UserID().Value(),
		Nickname:      target.Nickname,
		Role:          member.Role().String(),
		JoinedAt:      member.JoinedAt(),
		RoleChangedAt: now,
	}, nil
}
