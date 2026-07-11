package group

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
	appshared "github.com/training-judge-center/backend/internal/application/shared"
	domainGroup "github.com/training-judge-center/backend/internal/domain/group"
	"github.com/training-judge-center/backend/internal/domain/shared"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type AcceptInviteInput struct {
	GroupID      string
	InvitationID string
	CurrentUser  appshared.CurrentUser
}

type AcceptInviteOutput struct {
	Member MemberDTO
}

type AcceptInviteUseCase struct {
	groupRepo      domainGroup.Repository
	memberRepo     domainGroup.MemberRepository
	invitationRepo domainGroup.InvitationRepository
	txManager      appshared.TransactionManager
}

func NewAcceptInviteUseCase(
	groupRepo domainGroup.Repository,
	memberRepo domainGroup.MemberRepository,
	invitationRepo domainGroup.InvitationRepository,
	txManager appshared.TransactionManager,
) *AcceptInviteUseCase {
	return &AcceptInviteUseCase{
		groupRepo:      groupRepo,
		memberRepo:     memberRepo,
		invitationRepo: invitationRepo,
		txManager:      txManager,
	}
}

func (uc *AcceptInviteUseCase) Execute(ctx context.Context, input AcceptInviteInput) (*AcceptInviteOutput, error) {
	if input.InvitationID == "" {
		return nil, apperror.NewValidation([]apperror.FieldError{
			{Field: "invitationId", Message: "invitationId is required"},
		})
	}

	inv, err := uc.invitationRepo.FindByID(ctx, input.InvitationID)
	if err != nil {
		return nil, err
	}

	if inv.GroupID() != input.GroupID {
		return nil, apperror.NewNotFound(domainGroup.ErrCodeInvitationNotFound, "invitation not found")
	}

	if inv.HasInvitee() && inv.InviteeID().Value() != input.CurrentUser.ID {
		return nil, apperror.NewForbidden(ErrCodeInsufficientPermissions, "this invitation is not addressed to you")
	}

	if !inv.IsPending() {
		return nil, apperror.NewBadRequest(domainGroup.ErrCodeInvitationAlreadyProcessed, "this invitation has already been processed")
	}

	now := time.Now()
	if inv.IsExpired(now) {
		if err := uc.invitationRepo.TransitionStatus(ctx, inv.ID(), domainGroup.InvitationStatusPending, domainGroup.InvitationStatusExpired); err != nil {
			// Internal errors are already logged at the adapter boundary; only
			// log here when the CAS lost a race (RowsAffected==0), which the
			// adapter reports as a conflict without logging it itself — leaving
			// this case silent would make the invitation's stuck PENDING state
			// undiagnosable.
			if ae, ok := err.(*apperror.AppError); !ok || ae.Kind != apperror.KindInternal {
				slog.ErrorContext(ctx, "failed to mark invitation as expired", "invitation_id", inv.ID(), "error", err)
			}
		}
		return nil, apperror.NewBadRequest(domainGroup.ErrCodeInvitationExpired, "this invitation has expired")
	}

	if err := inv.Accept(); err != nil {
		return nil, err
	}

	userID := shared.RestoreUserID(input.CurrentUser.ID)
	var member *domainGroup.GroupMember

	err = uc.txManager.WithTx(ctx, func(txCtx context.Context) error {
		// Re-read the invitation inside the transaction: a general
		// (invitee-less) invitation never transitions status on accept, so
		// the earlier IsPending() check alone would miss a concurrent Revoke
		// that committed between that check and this transaction. Per-invitee
		// invitations get the same protection here as a bonus — their
		// TransitionStatus call below is still the authoritative compare-and-swap.
		currentInv, err := uc.invitationRepo.FindByID(txCtx, inv.ID())
		if err != nil {
			return err
		}
		if !currentInv.IsPending() {
			return apperror.NewConflict(domainGroup.ErrCodeInvitationAlreadyProcessed, "this invitation has already been processed")
		}

		g, err := uc.groupRepo.FindByID(txCtx, inv.GroupID())
		if err != nil {
			return err
		}
		if g.JoinPolicy() != domainGroup.JoinPolicyInvite {
			return apperror.NewForbidden(ErrCodeInsufficientPermissions, "this group no longer accepts invitations")
		}

		existing, err := uc.memberRepo.FindByGroupAndUser(txCtx, inv.GroupID(), userID)
		if err != nil {
			return err
		}
		if existing != nil {
			return apperror.NewConflict(domainGroup.ErrCodeAlreadyMember, "you are already a member of this group")
		}

		newMember, err := domainGroup.NewGroupMember(uuid.New().String(), inv.GroupID(), userID, domainGroup.MemberRoleMember, domainGroup.JoinMethodInvitation, nil, now)
		if err != nil {
			return err
		}
		if err := uc.memberRepo.Save(txCtx, newMember); err != nil {
			return err
		}
		member = newMember

		// A general (invitee-less) invitation stays PENDING so it can be
		// reused by other users until it expires or is revoked — only a
		// per-invitee invitation is consumed on accept.
		if inv.HasInvitee() {
			if err := uc.invitationRepo.TransitionStatus(txCtx, inv.ID(), domainGroup.InvitationStatusPending, domainGroup.InvitationStatusAccepted); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return &AcceptInviteOutput{Member: memberToDTO(member)}, nil
}
