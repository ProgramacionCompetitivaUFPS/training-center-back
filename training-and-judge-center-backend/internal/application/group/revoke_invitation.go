package group

import (
	"context"

	appshared "github.com/training-judge-center/backend/internal/application/shared"
	domainGroup "github.com/training-judge-center/backend/internal/domain/group"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type RevokeInvitationInput struct {
	GroupID      string
	InvitationID string
	CurrentUser  appshared.CurrentUser
}

type RevokeInvitationUseCase struct {
	memberRepo     domainGroup.MemberRepository
	invitationRepo domainGroup.InvitationRepository
}

func NewRevokeInvitationUseCase(memberRepo domainGroup.MemberRepository, invitationRepo domainGroup.InvitationRepository) *RevokeInvitationUseCase {
	return &RevokeInvitationUseCase{memberRepo: memberRepo, invitationRepo: invitationRepo}
}

func (uc *RevokeInvitationUseCase) Execute(ctx context.Context, input RevokeInvitationInput) error {
	if err := requireLeadOrAdmin(ctx, uc.memberRepo, input.GroupID, input.CurrentUser); err != nil {
		return err
	}

	inv, err := uc.invitationRepo.FindByID(ctx, input.InvitationID)
	if err != nil {
		return err
	}
	if inv.GroupID() != input.GroupID {
		return apperror.NewNotFound(domainGroup.ErrCodeInvitationNotFound, "invitation not found")
	}

	if err := inv.Revoke(); err != nil {
		return err
	}

	return uc.invitationRepo.TransitionStatus(ctx, inv.ID(), domainGroup.InvitationStatusPending, domainGroup.InvitationStatusRevoked)
}
