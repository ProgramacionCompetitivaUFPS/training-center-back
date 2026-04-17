package group

import (
	"time"

	"github.com/training-judge-center/backend/internal/domain/shared"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type GroupMember struct {
	id       string
	groupID  string
	userID   shared.UserID
	role     MemberRole
	joinedAt time.Time
}

func NewGroupMember(id, groupID string, userID shared.UserID, role MemberRole) (*GroupMember, error) {
	if id == "" {
		return nil, apperror.NewBadRequest(apperror.ErrCodeBadRequest, "group member id cannot be empty")
	}
	if groupID == "" {
		return nil, apperror.NewBadRequest(apperror.ErrCodeBadRequest, "group id cannot be empty")
	}
	return &GroupMember{
		id:       id,
		groupID:  groupID,
		userID:   userID,
		role:     role,
		joinedAt: time.Now().UTC(),
	}, nil
}

func RestoreGroupMember(id, groupID string, userID shared.UserID, role MemberRole, joinedAt time.Time) *GroupMember {
	return &GroupMember{
		id:       id,
		groupID:  groupID,
		userID:   userID,
		role:     role,
		joinedAt: joinedAt,
	}
}

func (m *GroupMember) ID() string            { return m.id }
func (m *GroupMember) GroupID() string       { return m.groupID }
func (m *GroupMember) UserID() shared.UserID { return m.userID }
func (m *GroupMember) Role() MemberRole      { return m.role }
func (m *GroupMember) JoinedAt() time.Time   { return m.joinedAt }
func (m *GroupMember) IsLead() bool          { return m.role == MemberRoleLead }

func (m *GroupMember) Promote() { m.role = MemberRoleLead }
func (m *GroupMember) Demote()  { m.role = MemberRoleMember }
