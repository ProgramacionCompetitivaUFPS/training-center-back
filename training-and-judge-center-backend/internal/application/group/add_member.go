package group

import (
	"context"
	"time"

	"github.com/google/uuid"
	appshared "github.com/training-judge-center/backend/internal/application/shared"
	domainGroup "github.com/training-judge-center/backend/internal/domain/group"
	"github.com/training-judge-center/backend/internal/domain/shared"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type AddMemberInput struct {
	GroupID     string
	Nickname    string
	Role        string
	CurrentUser appshared.CurrentUser
}

type AddMemberOutput struct {
	GroupID    string
	UserID     string
	Nickname   string
	Role       string
	JoinedAt   time.Time
	AddedBy    string
	JoinMethod string
}

type AddMemberUseCase struct {
	groupRepo        domainGroup.Repository
	memberRepo       domainGroup.MemberRepository
	nicknameResolver NicknameResolver
}

func NewAddMemberUseCase(
	groupRepo domainGroup.Repository,
	memberRepo domainGroup.MemberRepository,
	nicknameResolver NicknameResolver,
) *AddMemberUseCase {
	return &AddMemberUseCase{
		groupRepo:        groupRepo,
		memberRepo:       memberRepo,
		nicknameResolver: nicknameResolver,
	}
}

func (uc *AddMemberUseCase) Execute(ctx context.Context, input AddMemberInput) (*AddMemberOutput, error) {
	var fieldErrs []apperror.FieldError
	if input.Nickname == "" {
		fieldErrs = append(fieldErrs, apperror.FieldError{Field: "nickname", Message: "nickname is required"})
	}
	if input.Role == "" {
		fieldErrs = append(fieldErrs, apperror.FieldError{Field: "role", Message: "role is required"})
	}
	if len(fieldErrs) > 0 {
		return nil, apperror.NewValidation(fieldErrs)
	}

	role, err := domainGroup.NewMemberRole(input.Role)
	if err != nil {
		return nil, err
	}

	g, err := uc.groupRepo.FindByID(ctx, input.GroupID)
	if err != nil {
		return nil, err
	}

	if err := requireLeadOrAdmin(ctx, uc.memberRepo, input.GroupID, input.CurrentUser); err != nil {
		return nil, err
	}

	if g.IsDefault() {
		return nil, apperror.NewBadRequest(domainGroup.ErrCodeCannotRemoveFromGlobalGroup, "cannot manually add members to the global group")
	}

	target, err := uc.nicknameResolver.ResolveByNickname(ctx, input.Nickname)
	if err != nil {
		return nil, err
	}
	if target == nil {
		return nil, apperror.NewNotFound("NICKNAME_NOT_FOUND", "the specified nickname does not exist")
	}

	if role == domainGroup.MemberRoleLead && target.SystemRole == shared.RoleContestant.String() {
		return nil, apperror.NewBadRequest(domainGroup.ErrCodeInvalidLeadAssignment, "only Coaches and Admins can be assigned as leads")
	}

	targetID := shared.RestoreUserID(target.ID)
	existing, err := uc.memberRepo.FindByGroupAndUser(ctx, input.GroupID, targetID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, apperror.NewConflict(domainGroup.ErrCodeAlreadyMember, "user is already a member of this group")
	}

	now := time.Now()
	callerID := shared.RestoreUserID(input.CurrentUser.ID)
	member, err := domainGroup.NewGroupMember(uuid.New().String(), input.GroupID, targetID, role, domainGroup.JoinMethodDirectAdd, &callerID, now)
	if err != nil {
		return nil, err
	}

	if err := uc.memberRepo.Save(ctx, member); err != nil {
		return nil, err
	}

	return &AddMemberOutput{
		GroupID:    member.GroupID(),
		UserID:     member.UserID().Value(),
		Nickname:   target.Nickname,
		Role:       member.Role().String(),
		JoinedAt:   member.JoinedAt(),
		AddedBy:    input.CurrentUser.ID,
		JoinMethod: member.JoinMethod().String(),
	}, nil
}
