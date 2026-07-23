package group

import (
	"context"

	"github.com/training-judge-center/backend/internal/domain/shared"
)

type InvitationFilters struct {
	Status *InvitationStatus
	Page   int
	Limit  int
}

type InvitationRepository interface {
	// Save inserts a new invitation. It is not an upsert — status transitions
	// go through TransitionStatus.
	Save(ctx context.Context, inv *GroupInvitation) error
	FindByID(ctx context.Context, id string) (*GroupInvitation, error)
	// FindPendingByGroupAndInvitee returns the PENDING invitation for this
	// (groupID, inviteeID) pair, if any. inviteeID == nil looks up the
	// general (link-only) invitation for the group instead.
	FindPendingByGroupAndInvitee(ctx context.Context, groupID string, inviteeID *shared.UserID) (*GroupInvitation, error)
	FindByGroup(ctx context.Context, groupID string, filters InvitationFilters) ([]*GroupInvitation, int, error)
	// TransitionStatus atomically moves an invitation from `from` to `to`
	// (UPDATE ... WHERE id = $1 AND status = $2). Returns a conflict error if
	// the row is no longer in `from` state — guards against two concurrent
	// transitions (e.g. accept vs. revoke) racing on the same row.
	TransitionStatus(ctx context.Context, id string, from, to InvitationStatus) error
}
