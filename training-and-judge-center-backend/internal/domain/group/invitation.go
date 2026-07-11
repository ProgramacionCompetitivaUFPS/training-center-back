package group

import (
	"time"

	"github.com/training-judge-center/backend/internal/domain/shared"
	"github.com/training-judge-center/backend/pkg/apperror"
)

// InvitationExpiryDuration is the fixed TTL for a group invitation, evaluated
// lazily (no background job) — see GroupInvitation.IsExpired.
const InvitationExpiryDuration = 72 * time.Hour

type GroupInvitation struct {
	id        string
	groupID   string
	inviteeID *shared.UserID // nil = general invitation (link-only, no specific invitee)
	invitedBy shared.UserID
	status    InvitationStatus
	expiresAt time.Time
	createdAt time.Time
}

func NewGroupInvitation(id, groupID string, inviteeID *shared.UserID, invitedBy shared.UserID, now time.Time) (*GroupInvitation, error) {
	if id == "" {
		return nil, apperror.NewInternal()
	}
	if groupID == "" {
		return nil, apperror.NewInternal()
	}
	if invitedBy.Value() == "" {
		return nil, apperror.NewInternal()
	}
	t := now.UTC()
	return &GroupInvitation{
		id:        id,
		groupID:   groupID,
		inviteeID: inviteeID,
		invitedBy: invitedBy,
		status:    InvitationStatusPending,
		expiresAt: t.Add(InvitationExpiryDuration),
		createdAt: t,
	}, nil
}

func RestoreGroupInvitation(id, groupID string, inviteeID *shared.UserID, invitedBy shared.UserID, status InvitationStatus, expiresAt, createdAt time.Time) *GroupInvitation {
	return &GroupInvitation{
		id:        id,
		groupID:   groupID,
		inviteeID: inviteeID,
		invitedBy: invitedBy,
		status:    status,
		expiresAt: expiresAt,
		createdAt: createdAt,
	}
}

func (i *GroupInvitation) ID() string                { return i.id }
func (i *GroupInvitation) GroupID() string           { return i.groupID }
func (i *GroupInvitation) InviteeID() *shared.UserID { return i.inviteeID }
func (i *GroupInvitation) InvitedBy() shared.UserID  { return i.invitedBy }
func (i *GroupInvitation) Status() InvitationStatus  { return i.status }
func (i *GroupInvitation) ExpiresAt() time.Time      { return i.expiresAt }
func (i *GroupInvitation) CreatedAt() time.Time      { return i.createdAt }

func (i *GroupInvitation) IsPending() bool              { return i.status == InvitationStatusPending }
func (i *GroupInvitation) HasInvitee() bool             { return i.inviteeID != nil }
func (i *GroupInvitation) IsExpired(now time.Time) bool { return now.After(i.expiresAt) }

func (i *GroupInvitation) Accept() error {
	if !i.IsPending() {
		return apperror.NewBadRequest(ErrCodeInvitationAlreadyProcessed, "this invitation has already been processed")
	}
	i.status = InvitationStatusAccepted
	return nil
}

func (i *GroupInvitation) Revoke() error {
	if !i.IsPending() {
		return apperror.NewBadRequest(ErrCodeInvitationAlreadyProcessed, "this invitation has already been processed")
	}
	i.status = InvitationStatusRevoked
	return nil
}

func (i *GroupInvitation) Expire() error {
	if !i.IsPending() {
		return apperror.NewBadRequest(ErrCodeInvitationAlreadyProcessed, "this invitation has already been processed")
	}
	i.status = InvitationStatusExpired
	return nil
}
