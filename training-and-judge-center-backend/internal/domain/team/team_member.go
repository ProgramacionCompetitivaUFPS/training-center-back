package team

import (
	"time"

	"github.com/training-judge-center/backend/internal/domain/shared"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type TeamMember struct {
	id       string
	teamID   string
	userID   shared.UserID
	joinedAt time.Time
}

func NewTeamMember(id, teamID string, userID shared.UserID, now time.Time) (*TeamMember, error) {
	if id == "" || teamID == "" {
		return nil, apperror.NewInternal()
	}
	if userID.Value() == "" {
		return nil, apperror.NewInternal()
	}
	return &TeamMember{
		id:       id,
		teamID:   teamID,
		userID:   userID,
		joinedAt: now.UTC(),
	}, nil
}

func RestoreTeamMember(id, teamID string, userID shared.UserID, joinedAt time.Time) *TeamMember {
	return &TeamMember{
		id:       id,
		teamID:   teamID,
		userID:   userID,
		joinedAt: joinedAt,
	}
}

func (m *TeamMember) ID() string            { return m.id }
func (m *TeamMember) TeamID() string        { return m.teamID }
func (m *TeamMember) UserID() shared.UserID { return m.userID }
func (m *TeamMember) JoinedAt() time.Time   { return m.joinedAt }
