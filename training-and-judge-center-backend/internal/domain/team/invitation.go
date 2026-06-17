package team

import (
	"time"

	"github.com/training-judge-center/backend/internal/domain/shared"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type TeamInvitation struct {
	id        string
	teamID    string
	inviteeID shared.UserID
	invitedBy shared.UserID
	createdAt time.Time
}

func NewTeamInvitation(id, teamID string, inviteeID, invitedBy shared.UserID, now time.Time) (*TeamInvitation, error) {
	if id == "" || teamID == "" {
		return nil, apperror.NewInternal()
	}
	if inviteeID.Value() == "" || invitedBy.Value() == "" {
		return nil, apperror.NewInternal()
	}
	return &TeamInvitation{
		id:        id,
		teamID:    teamID,
		inviteeID: inviteeID,
		invitedBy: invitedBy,
		createdAt: now.UTC(),
	}, nil
}

func RestoreTeamInvitation(id, teamID string, inviteeID, invitedBy shared.UserID, createdAt time.Time) *TeamInvitation {
	return &TeamInvitation{
		id:        id,
		teamID:    teamID,
		inviteeID: inviteeID,
		invitedBy: invitedBy,
		createdAt: createdAt,
	}
}

func (i *TeamInvitation) ID() string              { return i.id }
func (i *TeamInvitation) TeamID() string          { return i.teamID }
func (i *TeamInvitation) InviteeID() shared.UserID { return i.inviteeID }
func (i *TeamInvitation) InvitedBy() shared.UserID { return i.invitedBy }
func (i *TeamInvitation) CreatedAt() time.Time    { return i.createdAt }
