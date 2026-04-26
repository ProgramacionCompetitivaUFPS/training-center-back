package group_test

import (
	"testing"
	"time"

	"github.com/training-judge-center/backend/internal/domain/group"
	"github.com/training-judge-center/backend/internal/domain/shared"
	"github.com/training-judge-center/backend/pkg/apperror"
)

func testUserID() shared.UserID { return shared.RestoreUserID("user-abc") }

func newMember(t *testing.T, id, groupID string, role group.MemberRole) *group.GroupMember {
	t.Helper()
	m, err := group.NewGroupMember(id, groupID, testUserID(), role, nil)
	if err != nil {
		t.Fatalf("NewGroupMember: %v", err)
	}
	return m
}

func TestNewGroupMember_Valid(t *testing.T) {
	fixed := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	m, err := group.NewGroupMember("m-1", "g-1", testUserID(), group.MemberRoleMember,
		func() time.Time { return fixed })
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.ID() != "m-1" {
		t.Errorf("ID() = %q", m.ID())
	}
	if m.GroupID() != "g-1" {
		t.Errorf("GroupID() = %q", m.GroupID())
	}
	if m.UserID() != testUserID() {
		t.Errorf("UserID() = %v", m.UserID())
	}
	if m.Role() != group.MemberRoleMember {
		t.Errorf("Role() = %v", m.Role())
	}
	if !m.JoinedAt().Equal(fixed) {
		t.Errorf("JoinedAt() = %v, want %v", m.JoinedAt(), fixed)
	}
	if m.IsLead() {
		t.Error("IsLead() should be false for MEMBER role")
	}
}

func TestNewGroupMember_EmptyID(t *testing.T) {
	_, err := group.NewGroupMember("", "g-1", testUserID(), group.MemberRoleMember, nil)
	if err == nil {
		t.Fatal("expected error for empty id, got nil")
	}
}

func TestNewGroupMember_EmptyGroupID(t *testing.T) {
	_, err := group.NewGroupMember("m-1", "", testUserID(), group.MemberRoleMember, nil)
	if err == nil {
		t.Fatal("expected error for empty groupID, got nil")
	}
}

func TestNewGroupMember_EmptyUserID(t *testing.T) {
	zeroUser := shared.RestoreUserID("")
	_, err := group.NewGroupMember("m-1", "g-1", zeroUser, group.MemberRoleMember, nil)
	if err == nil {
		t.Fatal("expected error for zero-value userID, got nil")
	}
	appErr, ok := err.(*apperror.AppError)
	if !ok {
		t.Fatalf("expected *apperror.AppError, got %T", err)
	}
	if appErr.Code != apperror.ErrCodeInternalError {
		t.Errorf("Code = %q, want %q", appErr.Code, apperror.ErrCodeInternalError)
	}
}

func TestGroupMember_IsLead(t *testing.T) {
	lead := newMember(t, "m-1", "g-1", group.MemberRoleLead)
	if !lead.IsLead() {
		t.Error("IsLead() should be true for LEAD role")
	}
}

func TestGroupMember_Promote(t *testing.T) {
	m := newMember(t, "m-1", "g-1", group.MemberRoleMember)
	if err := m.Promote(); err != nil {
		t.Fatalf("Promote() unexpected error: %v", err)
	}
	if m.Role() != group.MemberRoleLead {
		t.Errorf("Role() after Promote = %v, want LEAD", m.Role())
	}
}

func TestGroupMember_Promote_AlreadyLead(t *testing.T) {
	m := newMember(t, "m-1", "g-1", group.MemberRoleLead)
	err := m.Promote()
	if err == nil {
		t.Fatal("Promote() on already-LEAD expected error, got nil")
	}
	appErr, ok := err.(*apperror.AppError)
	if !ok {
		t.Fatalf("expected *apperror.AppError, got %T", err)
	}
	if appErr.Code != group.ErrCodeRoleUnchanged {
		t.Errorf("Code = %q, want %q", appErr.Code, group.ErrCodeRoleUnchanged)
	}
}

func TestGroupMember_Demote(t *testing.T) {
	m := newMember(t, "m-1", "g-1", group.MemberRoleLead)
	if err := m.Demote(); err != nil {
		t.Fatalf("Demote() unexpected error: %v", err)
	}
	if m.Role() != group.MemberRoleMember {
		t.Errorf("Role() after Demote = %v, want MEMBER", m.Role())
	}
}

func TestGroupMember_Demote_AlreadyMember(t *testing.T) {
	m := newMember(t, "m-1", "g-1", group.MemberRoleMember)
	err := m.Demote()
	if err == nil {
		t.Fatal("Demote() on already-MEMBER expected error, got nil")
	}
	appErr, ok := err.(*apperror.AppError)
	if !ok {
		t.Fatalf("expected *apperror.AppError, got %T", err)
	}
	if appErr.Code != group.ErrCodeRoleUnchanged {
		t.Errorf("Code = %q, want %q", appErr.Code, group.ErrCodeRoleUnchanged)
	}
}

func TestRestoreGroupMember(t *testing.T) {
	joined := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	m := group.RestoreGroupMember("m-99", "g-99", testUserID(), group.MemberRoleLead, joined)
	if m.ID() != "m-99" {
		t.Errorf("ID() = %q", m.ID())
	}
	if m.GroupID() != "g-99" {
		t.Errorf("GroupID() = %q", m.GroupID())
	}
	if !m.JoinedAt().Equal(joined) {
		t.Errorf("JoinedAt() = %v, want %v", m.JoinedAt(), joined)
	}
	if !m.IsLead() {
		t.Error("IsLead() should be true")
	}
}
