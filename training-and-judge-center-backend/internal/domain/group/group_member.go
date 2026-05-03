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
	removedAt  *time.Time
}

// NewGroupMember constructs a validated active GroupMember.
// Pass a non-nil clock for deterministic joinedAt in tests; nil defaults to time.Now.
func NewGroupMember(
	id, groupID string,
	userID shared.UserID,
	role MemberRole,
	addedBy *shared.UserID,
	joinMethod JoinMethod,
	clock func() time.Time,
) (*GroupMember, error) {
	if id == "" || groupID == "" || userID.Value() == "" {
		return nil, apperror.NewInternal()
	}
	if _, err := NewJoinMethod(string(joinMethod)); err != nil {
		return nil, err
	}
	if joinMethod == JoinMethodDirectAdd && addedBy == nil {
		return nil, apperror.NewInternal()
	}
	if joinMethod == JoinMethodOpenJoin && addedBy != nil {
		return nil, apperror.NewInternal()
	}
	if clock == nil {
		clock = time.Now
	}
	return &GroupMember{
		id:         id,
		groupID:    groupID,
		userID:     userID,
		role:       role,
		joinedAt:   clock().UTC(),
		addedBy:    addedBy,
		joinMethod: joinMethod,
	}, nil
}

func RestoreGroupMember(
	id, groupID string,
	userID shared.UserID,
	role MemberRole,
	joinedAt time.Time,
	addedBy *shared.UserID,
	joinMethod JoinMethod,
	removedAt *time.Time,
) *GroupMember {
	return &GroupMember{
		id:         id,
		groupID:    groupID,
		userID:     userID,
		role:       role,
		joinedAt:   joinedAt,
		addedBy:    addedBy,
		joinMethod: joinMethod,
		removedAt:  removedAt,
	}
}

func (m *GroupMember) ID() string              { return m.id }
func (m *GroupMember) GroupID() string         { return m.groupID }
func (m *GroupMember) UserID() shared.UserID   { return m.userID }
func (m *GroupMember) Role() MemberRole        { return m.role }
func (m *GroupMember) JoinedAt() time.Time     { return m.joinedAt }
func (m *GroupMember) AddedBy() *shared.UserID { return m.addedBy }
func (m *GroupMember) JoinMethod() JoinMethod  { return m.joinMethod }
func (m *GroupMember) RemovedAt() *time.Time   { return m.removedAt }
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
