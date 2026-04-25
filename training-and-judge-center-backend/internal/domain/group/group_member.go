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

func NewGroupMember(id, groupID string, userID shared.UserID, role MemberRole, clock func() time.Time) (*GroupMember, error) {
	if id == "" {
		return nil, apperror.NewBadRequest(apperror.ErrCodeBadRequest, "group member id cannot be empty")
	}
	if groupID == "" {
		return nil, apperror.NewBadRequest(apperror.ErrCodeBadRequest, "group id cannot be empty")
	}
	if userID.Value() == "" {
		return nil, apperror.NewBadRequest(apperror.ErrCodeBadRequest, "user id cannot be empty")
	}
	if clock == nil {
		clock = time.Now
	}
	return &GroupMember{
		id:       id,
		groupID:  groupID,
		userID:   userID,
		role:     role,
		joinedAt: clock().UTC(),
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

func (m *GroupMember) Promote() error {
	if m.role == MemberRoleLead {
		return apperror.NewConflict(ErrCodeRoleUnchanged, "member is already a lead")
	}
	m.role = MemberRoleLead
	return nil
}

func (m *GroupMember) Demote() error {
	if m.role == MemberRoleMember {
		return apperror.NewConflict(ErrCodeRoleUnchanged, "member is already a regular member")
	}
	m.role = MemberRoleMember
	return nil
}
