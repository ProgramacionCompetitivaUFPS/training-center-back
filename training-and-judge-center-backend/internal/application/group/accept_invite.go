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

type AcceptInviteInput struct {
	Token       string
	CurrentUser appshared.CurrentUser
}

type AcceptInviteOutput struct {
	Member *domainGroup.GroupMember
}

type AcceptInviteUseCase struct {
	groupRepo     domainGroup.Repository
	memberRepo    domainGroup.MemberRepository
	invitationSvc InvitationTokenService
}

func NewAcceptInviteUseCase(
	groupRepo domainGroup.Repository,
	memberRepo domainGroup.MemberRepository,
	invitationSvc InvitationTokenService,
) *AcceptInviteUseCase {
	return &AcceptInviteUseCase{
		groupRepo:     groupRepo,
		memberRepo:    memberRepo,
		invitationSvc: invitationSvc,
	}
}

func (uc *AcceptInviteUseCase) Execute(ctx context.Context, input AcceptInviteInput) (*AcceptInviteOutput, error) {
	if input.Token == "" {
		return nil, apperror.NewValidation([]apperror.FieldError{
			{Field: "token", Message: "token is required"},
		})
	}

	claims, err := uc.invitationSvc.ValidateInviteToken(input.Token)
	if err != nil {
		return nil, err
	}

	g, err := uc.groupRepo.FindByID(ctx, claims.GroupID)
	if err != nil {
		return nil, err
	}

	if g.JoinPolicy() != domainGroup.JoinPolicyInvite {
		return nil, apperror.NewForbidden(ErrCodeInsufficientPermissions, "this group no longer accepts invitations")
	}

	userID := shared.RestoreUserID(input.CurrentUser.ID)
	existing, err := uc.memberRepo.FindByGroupAndUser(ctx, claims.GroupID, userID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, apperror.NewConflict(ErrCodeAlreadyMember, "you are already a member of this group")
	}

	now := time.Now()
	newMemberID := uuid.New().String()
	member, err := domainGroup.NewGroupMember(newMemberID, claims.GroupID, userID, domainGroup.MemberRoleMember, now)
	if err != nil {
		return nil, err
	}

	if err := uc.memberRepo.Save(ctx, member); err != nil {
		return nil, err
	}

	return &AcceptInviteOutput{Member: member}, nil
}
