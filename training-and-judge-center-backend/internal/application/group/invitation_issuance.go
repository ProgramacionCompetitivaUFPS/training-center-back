package group

import (
	"context"
	"time"

	"github.com/google/uuid"
	domainGroup "github.com/training-judge-center/backend/internal/domain/group"
	"github.com/training-judge-center/backend/internal/domain/shared"
	"github.com/training-judge-center/backend/pkg/apperror"
)

// issueInvitation creates a new pending invitation for inviteeID in groupID,
// revoking any existing pending one for the same invitee first. Must be
// called from inside the caller's transaction (txCtx): it re-reads the group
// and re-checks JoinPolicy itself, guarding against a concurrent UpdateGroup
// that switched the policy away from INVITE between the caller's initial
// check and this transaction's commit.
func issueInvitation(
	txCtx context.Context,
	groupRepo domainGroup.Repository,
	invitationRepo domainGroup.InvitationRepository,
	groupID string,
	inviteeID *shared.UserID,
	invitedBy shared.UserID,
	now time.Time,
) (*domainGroup.GroupInvitation, error) {
	currentGroup, err := groupRepo.FindByID(txCtx, groupID)
	if err != nil {
		return nil, err
	}
	if currentGroup.JoinPolicy() != domainGroup.JoinPolicyInvite {
		return nil, apperror.NewBadRequest(ErrCodeInvalidJoinPolicy, "this group does not use invite mode")
	}

	existing, err := invitationRepo.FindPendingByGroupAndInvitee(txCtx, groupID, inviteeID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		if err := existing.Revoke(); err != nil {
			return nil, err
		}
		if err := invitationRepo.TransitionStatus(txCtx, existing.ID(), domainGroup.InvitationStatusPending, domainGroup.InvitationStatusRevoked); err != nil {
			return nil, err
		}
	}

	inv, err := domainGroup.NewGroupInvitation(uuid.New().String(), groupID, inviteeID, invitedBy, now)
	if err != nil {
		return nil, err
	}
	if err := invitationRepo.Save(txCtx, inv); err != nil {
		return nil, err
	}
	return inv, nil
}
