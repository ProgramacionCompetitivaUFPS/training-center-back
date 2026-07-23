package team

import (
	"context"

	appShared "github.com/training-judge-center/backend/internal/application/shared"
	domainTeam "github.com/training-judge-center/backend/internal/domain/team"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type RejectInvitationInput struct {
	CurrentUser  appShared.CurrentUser
	InvitationID string
}

type RejectInvitationUseCase struct {
	invitationRepo domainTeam.InvitationRepository
}

func NewRejectInvitationUseCase(invitationRepo domainTeam.InvitationRepository) *RejectInvitationUseCase {
	return &RejectInvitationUseCase{invitationRepo: invitationRepo}
}

func (uc *RejectInvitationUseCase) Execute(ctx context.Context, in RejectInvitationInput) error {
	inv, err := uc.invitationRepo.FindByID(ctx, in.InvitationID)
	if err != nil {
		return err
	}

	if inv.InviteeID().Value() != in.CurrentUser.ID {
		return apperror.NewForbidden(domainTeam.ErrCodeNotYourInvitation, "This invitation was not sent to you")
	}

	return uc.invitationRepo.Delete(ctx, inv.ID())
}
