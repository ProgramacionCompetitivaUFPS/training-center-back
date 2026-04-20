package group_test

import (
	"testing"
	"time"

	"github.com/training-judge-center/backend/internal/domain/group"
	"github.com/training-judge-center/backend/internal/domain/shared"
	"github.com/training-judge-center/backend/pkg/apperror"
)

func newGroupName(t *testing.T, s string) group.GroupName {
	t.Helper()
	n, err := group.NewGroupName(s)
	if err != nil {
		t.Fatalf("NewGroupName(%q): %v", s, err)
	}
	return n
}

func testCreator() shared.UserID {
	return shared.RestoreUserID("user-123")
}

func TestNewGroup_Valid(t *testing.T) {
	name := newGroupName(t, "Algorithms 2025")
	g, err := group.NewGroup("g-1", name, nil, group.VisibilityVisible, group.JoinPolicyRequest, testCreator(), nil)
	if err != nil {
		t.Fatalf("NewGroup unexpected error: %v", err)
	}
	if g.ID() != "g-1" {
		t.Errorf("ID() = %q, want %q", g.ID(), "g-1")
	}
	if g.Name().Value() != "Algorithms 2025" {
		t.Errorf("Name() = %q, want %q", g.Name().Value(), "Algorithms 2025")
	}
	if g.IsDefault() {
		t.Error("IsDefault() should be false for a regular group")
	}
	if g.CreatedBy() != testCreator() {
		t.Errorf("CreatedBy() = %v, want %v", g.CreatedBy(), testCreator())
	}
	if g.Description() != nil {
		t.Error("Description should be nil when not provided")
	}
}

func TestNewGroup_ClockInjection(t *testing.T) {
	fixed := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	name := newGroupName(t, "Timed Group")
	g, err := group.NewGroup("g-clock", name, nil, group.VisibilityVisible, group.JoinPolicyOpen, testCreator(),
		func() time.Time { return fixed })
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !g.CreatedAt().Equal(fixed) {
		t.Errorf("CreatedAt() = %v, want %v", g.CreatedAt(), fixed)
	}
	if !g.UpdatedAt().Equal(fixed) {
		t.Errorf("UpdatedAt() = %v, want %v", g.UpdatedAt(), fixed)
	}
}

func TestNewGroup_EmptyID(t *testing.T) {
	name := newGroupName(t, "No ID Group")
	_, err := group.NewGroup("", name, nil, group.VisibilityVisible, group.JoinPolicyOpen, testCreator(), nil)
	if err == nil {
		t.Fatal("NewGroup with empty id expected error, got nil")
	}
	appErr, ok := err.(*apperror.AppError)
	if !ok {
		t.Fatalf("expected *apperror.AppError, got %T", err)
	}
	if appErr.Code != apperror.ErrCodeBadRequest {
		t.Errorf("Code = %q, want %q", appErr.Code, apperror.ErrCodeBadRequest)
	}
}

func TestNewGroup_WithDescription(t *testing.T) {
	name := newGroupName(t, "My Group")
	desc := "A great group"
	g, err := group.NewGroup("g-1", name, &desc, group.VisibilityVisible, group.JoinPolicyOpen, testCreator(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if g.Description() == nil || *g.Description() != desc {
		t.Errorf("Description() = %v, want %q", g.Description(), desc)
	}
}

func TestNewGroup_InvalidPolicyCombination(t *testing.T) {
	name := newGroupName(t, "Private Group")
	cases := []struct {
		visibility group.Visibility
		joinPolicy group.JoinPolicy
	}{
		{group.VisibilityNotVisible, group.JoinPolicyOpen},
		{group.VisibilityNotVisible, group.JoinPolicyRequest},
	}
	for _, tc := range cases {
		_, err := group.NewGroup("g-1", name, nil, tc.visibility, tc.joinPolicy, testCreator(), nil)
		if err == nil {
			t.Errorf("NewGroup(%v, %v) expected INVALID_POLICY_COMBINATION error, got nil", tc.visibility, tc.joinPolicy)
		}
	}
}

func TestNewGroup_ValidNotVisibleInvite(t *testing.T) {
	name := newGroupName(t, "Secret Group")
	_, err := group.NewGroup("g-1", name, nil, group.VisibilityNotVisible, group.JoinPolicyInvite, testCreator(), nil)
	if err != nil {
		t.Errorf("NewGroup(NOT_VISIBLE, INVITE) unexpected error: %v", err)
	}
}

func TestGroup_CanBeDeleted(t *testing.T) {
	name := newGroupName(t, "Regular Group")
	g, _ := group.NewGroup("g-1", name, nil, group.VisibilityVisible, group.JoinPolicyOpen, testCreator(), nil)
	if !g.CanBeDeleted() {
		t.Error("regular group should be deletable")
	}

	gDefault := group.RestoreGroup("g-default", group.RestoreGroupName("global"), nil,
		group.VisibilityVisible, group.JoinPolicyInvite, true, testCreator(), time.Now(), time.Now())
	if gDefault.CanBeDeleted() {
		t.Error("default group should NOT be deletable")
	}
}

func TestGroup_UpdateMetadata(t *testing.T) {
	name := newGroupName(t, "Original Name")
	g, _ := group.NewGroup("g-1", name, nil, group.VisibilityVisible, group.JoinPolicyRequest, testCreator(), nil)
	fixedTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	g.WithClock(func() time.Time { return fixedTime })

	newName := newGroupName(t, "Updated Name")
	desc := "New description"
	descPtr := &desc
	g.UpdateMetadata(&newName, &descPtr)

	if g.Name().Value() != "Updated Name" {
		t.Errorf("Name() = %q, want %q", g.Name().Value(), "Updated Name")
	}
	if g.Description() == nil || *g.Description() != desc {
		t.Error("Description not updated")
	}
	if !g.UpdatedAt().Equal(fixedTime) {
		t.Errorf("UpdatedAt() = %v, want %v", g.UpdatedAt(), fixedTime)
	}
}

func TestGroup_UpdateMetadata_PartialUpdate(t *testing.T) {
	name := newGroupName(t, "My Group")
	desc := "original desc"
	g, _ := group.NewGroup("g-1", name, &desc, group.VisibilityVisible, group.JoinPolicyOpen, testCreator(), nil)

	newName := newGroupName(t, "New Name")
	g.UpdateMetadata(&newName, nil) // nil **string = no tocar description

	if g.Name().Value() != "New Name" {
		t.Errorf("Name should be updated")
	}
	if g.Description() == nil || *g.Description() != desc {
		t.Error("Description should remain unchanged when nil passed")
	}
}

func TestGroup_UpdateMetadata_ClearDescription(t *testing.T) {
	name := newGroupName(t, "My Group")
	desc := "will be cleared"
	g, _ := group.NewGroup("g-1", name, &desc, group.VisibilityVisible, group.JoinPolicyOpen, testCreator(), nil)

	var nilStr *string
	g.UpdateMetadata(nil, &nilStr) // &nilStr = limpiar description

	if g.Description() != nil {
		t.Errorf("Description should be nil after clearing, got %q", *g.Description())
	}
}

func TestGroup_UpdatePolicies_NilNilNoOp(t *testing.T) {
	name := newGroupName(t, "My Group")
	g, _ := group.NewGroup("g-1", name, nil, group.VisibilityVisible, group.JoinPolicyOpen, testCreator(), nil)
	before := g.UpdatedAt()

	fixed := before.Add(time.Hour)
	g.WithClock(func() time.Time { return fixed })

	if err := g.UpdatePolicies(nil, nil); err != nil {
		t.Fatalf("UpdatePolicies(nil, nil) unexpected error: %v", err)
	}
	if !g.UpdatedAt().Equal(before) {
		t.Errorf("UpdatedAt() changed on nil/nil no-op: got %v, want %v", g.UpdatedAt(), before)
	}
}

func TestGroup_UpdatePolicies_InvalidCombination(t *testing.T) {
	name := newGroupName(t, "My Group")
	g, _ := group.NewGroup("g-1", name, nil, group.VisibilityVisible, group.JoinPolicyOpen, testCreator(), nil)

	notVisible := group.VisibilityNotVisible
	open := group.JoinPolicyOpen
	err := g.UpdatePolicies(&notVisible, &open)
	if err == nil {
		t.Error("UpdatePolicies(NOT_VISIBLE, OPEN) expected error, got nil")
	}
}

func TestGroup_UpdatePolicies_ValidChangeJoinPolicy(t *testing.T) {
	name := newGroupName(t, "My Group")
	g, _ := group.NewGroup("g-1", name, nil, group.VisibilityVisible, group.JoinPolicyOpen, testCreator(), nil)

	invite := group.JoinPolicyInvite
	err := g.UpdatePolicies(nil, &invite)
	if err != nil {
		t.Errorf("UpdatePolicies(nil, INVITE) unexpected error: %v", err)
	}
	if g.JoinPolicy() != group.JoinPolicyInvite {
		t.Errorf("JoinPolicy() = %v, want INVITE", g.JoinPolicy())
	}
}

func TestGroup_UpdatePolicies_ValidChangeVisibility(t *testing.T) {
	name := newGroupName(t, "My Group")
	g, _ := group.NewGroup("g-1", name, nil, group.VisibilityVisible, group.JoinPolicyOpen, testCreator(), nil)

	notVisible := group.VisibilityNotVisible
	invite := group.JoinPolicyInvite
	err := g.UpdatePolicies(&notVisible, &invite)
	if err != nil {
		t.Errorf("UpdatePolicies(NOT_VISIBLE, INVITE) unexpected error: %v", err)
	}
	if g.Visibility() != group.VisibilityNotVisible {
		t.Errorf("Visibility() = %v, want NOT_VISIBLE", g.Visibility())
	}
}

func TestRestoreGroup(t *testing.T) {
	createdAt := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	updatedAt := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	desc := "restored description"

	g := group.RestoreGroup("g-99", group.RestoreGroupName("My Group"), &desc,
		group.VisibilityNotVisible, group.JoinPolicyInvite, false, testCreator(), createdAt, updatedAt)

	if g.ID() != "g-99" {
		t.Errorf("ID() = %q", g.ID())
	}
	if g.Name().Value() != "My Group" {
		t.Errorf("Name() = %q", g.Name().Value())
	}
	if g.IsDefault() {
		t.Error("IsDefault should be false")
	}
	if !g.CreatedAt().Equal(createdAt) {
		t.Errorf("CreatedAt() = %v", g.CreatedAt())
	}
}
