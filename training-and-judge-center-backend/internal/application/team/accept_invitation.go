package team

import (
	"context"
	"time"

	"github.com/google/uuid"
	appShared "github.com/training-judge-center/backend/internal/application/shared"
	domainShared "github.com/training-judge-center/backend/internal/domain/shared"
	domainTeam "github.com/training-judge-center/backend/internal/domain/team"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type AcceptInvitationInput struct {
	CurrentUser  appShared.CurrentUser
	InvitationID string
}

type AcceptInvitationOutput struct {
	JoinedAt time.Time
}

type AcceptInvitationUseCase struct {
	invitationRepo domainTeam.InvitationRepository
	memberRepo     domainTeam.MemberRepository
	txManager      appShared.TransactionManager
}

func NewAcceptInvitationUseCase(
	invitationRepo domainTeam.InvitationRepository,
	memberRepo domainTeam.MemberRepository,
	txManager appShared.TransactionManager,
) *AcceptInvitationUseCase {
	return &AcceptInvitationUseCase{
		invitationRepo: invitationRepo,
		memberRepo:     memberRepo,
		txManager:      txManager,
	}
}

func (uc *AcceptInvitationUseCase) Execute(ctx context.Context, in AcceptInvitationInput, now time.Time) (*AcceptInvitationOutput, error) {
	inv, err := uc.invitationRepo.FindByID(ctx, in.InvitationID)
	if err != nil {
		return nil, err
	}

	if inv.InviteeID().Value() != in.CurrentUser.ID {
		return nil, apperror.NewForbidden(domainTeam.ErrCodeNotYourInvitation, "This invitation was not sent to you")
	}

	member, err := domainTeam.NewTeamMember(uuid.New().String(), inv.TeamID(), domainShared.RestoreUserID(in.CurrentUser.ID), now)
	if err != nil {
		return nil, err
	}

	if err := uc.txManager.WithTx(ctx, func(txCtx context.Context) error {
		if err := uc.memberRepo.Save(txCtx, member); err != nil {
			return err
		}
		return uc.invitationRepo.Delete(txCtx, inv.ID())
	}); err != nil {
		return nil, err
	}

	return &AcceptInvitationOutput{JoinedAt: member.JoinedAt()}, nil
}
