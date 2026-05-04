package group

import (
	"context"

	domainGroup "github.com/training-judge-center/backend/internal/domain/group"
	"github.com/training-judge-center/backend/internal/application/shared"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type GenerateInviteInput struct {
	GroupID     string
	CurrentUser shared.CurrentUser
}

type GenerateInviteOutput struct {
	Token string
}

type GenerateInviteUseCase struct {
	groupRepo     domainGroup.Repository
	memberRepo    domainGroup.MemberRepository
	invitationSvc InvitationTokenService
}

func NewGenerateInviteUseCase(
	groupRepo domainGroup.Repository,
	memberRepo domainGroup.MemberRepository,
	invitationSvc InvitationTokenService,
) *GenerateInviteUseCase {
	return &GenerateInviteUseCase{
		groupRepo:     groupRepo,
		memberRepo:    memberRepo,
		invitationSvc: invitationSvc,
	}
}

func (uc *GenerateInviteUseCase) Execute(ctx context.Context, input GenerateInviteInput) (*GenerateInviteOutput, error) {
	if input.GroupID == "" {
		return nil, apperror.NewValidation([]apperror.FieldError{
			{Field: "groupId", Message: "groupId is required"},
		})
	}

	g, err := uc.groupRepo.FindByID(ctx, input.GroupID)
	if err != nil {
		return nil, err
	}

	if err := requireLeadOrAdmin(ctx, uc.memberRepo, input.GroupID, input.CurrentUser); err != nil {
		return nil, err
	}

	if g.JoinPolicy() != domainGroup.JoinPolicyInvite {
		return nil, apperror.NewForbidden(domainGroup.ErrCodeInsufficientPermissions, "this group does not use invite mode")
	}

	token, err := uc.invitationSvc.GenerateInviteToken(input.GroupID, input.CurrentUser.ID)
	if err != nil {
		return nil, err
	}

	return &GenerateInviteOutput{Token: token}, nil
}
