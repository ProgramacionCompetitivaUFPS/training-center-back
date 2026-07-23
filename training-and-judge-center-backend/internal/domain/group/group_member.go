package group

import (
	"time"

	"github.com/training-judge-center/backend/internal/domain/shared"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type GroupMember struct {
	id         string
	groupID    string
	userID     shared.UserID
	role       MemberRole
	joinedAt   time.Time
	addedBy    *shared.UserID
	joinMethod JoinMethod
}

func NewGroupMember(id, groupID string, userID shared.UserID, role MemberRole, joinMethod JoinMethod, addedBy *shared.UserID, now time.Time) (*GroupMember, error) {
	if id == "" {
		return nil, apperror.NewInternal()
	}
	if groupID == "" {
		return nil, apperror.NewInternal()
	}
	if userID.Value() == "" {
		return nil, apperror.NewInternal()
	}
	return &GroupMember{
		id:         id,
		groupID:    groupID,
		userID:     userID,
		role:       role,
		joinedAt:   now.UTC(),
		addedBy:    addedBy,
		joinMethod: joinMethod,
	}, nil
}

func RestoreGroupMember(id, groupID string, userID shared.UserID, role MemberRole, joinedAt time.Time, addedBy *shared.UserID, joinMethod JoinMethod) *GroupMember {
	return &GroupMember{
		id:         id,
		groupID:    groupID,
		userID:     userID,
		role:       role,
		joinedAt:   joinedAt,
		addedBy:    addedBy,
		joinMethod: joinMethod,
	}
}

func (m *GroupMember) ID() string              { return m.id }
func (m *GroupMember) GroupID() string         { return m.groupID }
func (m *GroupMember) UserID() shared.UserID   { return m.userID }
func (m *GroupMember) Role() MemberRole        { return m.role }
func (m *GroupMember) JoinedAt() time.Time     { return m.joinedAt }
func (m *GroupMember) AddedBy() *shared.UserID { return m.addedBy }
func (m *GroupMember) JoinMethod() JoinMethod  { return m.joinMethod }
func (m *GroupMember) IsLead() bool            { return m.role == MemberRoleLead }

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
