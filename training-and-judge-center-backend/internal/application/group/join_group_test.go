package group

import (
	"context"
	"errors"
	"testing"
	"time"

	domainGroup "github.com/training-judge-center/backend/internal/domain/group"
	"github.com/training-judge-center/backend/internal/domain/shared"
	"github.com/training-judge-center/backend/pkg/apperror"
)

func openGroup(t *testing.T) *domainGroup.Group {
	t.Helper()
	return mustGroup(t, "g1", "Open Club", domainGroup.VisibilityVisible, domainGroup.JoinPolicyOpen)
}

func TestJoinGroup_EmptyGroupIDReturnsValidationError(t *testing.T) {
	uc := NewJoinGroupUseCase(&fakeRepo{}, &fakeMemberRepo{})

	_, err := uc.Execute(context.Background(), JoinGroupInput{
		GroupID:     "",
		CurrentUser: currentUser("u1", shared.RoleContestant),
	})

	ae, ok := err.(*apperror.AppError)
	if !ok || ae.Code != apperror.ErrCodeValidationError {
		t.Fatalf("expected VALIDATION_ERROR, got %v", err)
	}
	if len(ae.Details) == 0 || ae.Details[0].Field != "groupId" {
		t.Errorf("expected field error on 'groupId', got %+v", ae.Details)
	}
}

func TestJoinGroup_GroupNotFoundReturns404(t *testing.T) {
	uc := NewJoinGroupUseCase(&fakeRepo{}, &fakeMemberRepo{})

	_, err := uc.Execute(context.Background(), JoinGroupInput{
		GroupID:     "nonexistent",
		CurrentUser: currentUser("u1", shared.RoleContestant),
	})

	ae, ok := err.(*apperror.AppError)
	if !ok || ae.Code != domainGroup.ErrCodeGroupNotFound {
		t.Fatalf("expected GROUP_NOT_FOUND, got %v", err)
	}
}

func TestJoinGroup_InvitePolicyReturns403(t *testing.T) {
	g := mustGroup(t, "g1", "Invite Club", domainGroup.VisibilityVisible, domainGroup.JoinPolicyInvite)
	uc := NewJoinGroupUseCase(&fakeRepo{groups: []*domainGroup.Group{g}}, &fakeMemberRepo{})

	_, err := uc.Execute(context.Background(), JoinGroupInput{
		GroupID:     "g1",
		CurrentUser: currentUser("u1", shared.RoleContestant),
	})

	ae, ok := err.(*apperror.AppError)
	if !ok || ae.Code != domainGroup.ErrCodeInsufficientPermissions {
		t.Fatalf("expected INSUFFICIENT_PERMISSIONS, got %v", err)
	}
	if ae.StatusCode != 403 {
		t.Errorf("expected HTTP 403, got %d", ae.StatusCode)
	}
}

func TestJoinGroup_RequestPolicyReturns403(t *testing.T) {
	g := mustGroup(t, "g1", "Request Club", domainGroup.VisibilityVisible, domainGroup.JoinPolicyRequest)
	uc := NewJoinGroupUseCase(&fakeRepo{groups: []*domainGroup.Group{g}}, &fakeMemberRepo{})

	_, err := uc.Execute(context.Background(), JoinGroupInput{
		GroupID:     "g1",
		CurrentUser: currentUser("u1", shared.RoleContestant),
	})

	ae, ok := err.(*apperror.AppError)
	if !ok || ae.Code != domainGroup.ErrCodeInsufficientPermissions {
		t.Fatalf("expected INSUFFICIENT_PERMISSIONS, got %v", err)
	}
}

func TestJoinGroup_AlreadyMemberReturns409(t *testing.T) {
	g := openGroup(t)
	userID := shared.RestoreUserID("u1")
	existingMember := domainGroup.RestoreGroupMember("m1", "g1", userID, domainGroup.MemberRoleMember, time.Now())
	memberRepo := &fakeMemberRepo{
		memberships: map[string]*domainGroup.GroupMember{
			keyOf("g1", userID): existingMember,
		},
	}
	uc := NewJoinGroupUseCase(&fakeRepo{groups: []*domainGroup.Group{g}}, memberRepo)

	_, err := uc.Execute(context.Background(), JoinGroupInput{
		GroupID:     "g1",
		CurrentUser: currentUser("u1", shared.RoleContestant),
	})

	ae, ok := err.(*apperror.AppError)
	if !ok || ae.Code != domainGroup.ErrCodeAlreadyMember {
		t.Fatalf("expected ALREADY_MEMBER, got %v", err)
	}
	if ae.StatusCode != 409 {
		t.Errorf("expected HTTP 409, got %d", ae.StatusCode)
	}
}

func TestJoinGroup_SuccessSavesMemberWithRoleMember(t *testing.T) {
	g := openGroup(t)
	memberRepo := &fakeMemberRepo{}
	uc := NewJoinGroupUseCase(&fakeRepo{groups: []*domainGroup.Group{g}}, memberRepo)

	out, err := uc.Execute(context.Background(), JoinGroupInput{
		GroupID:     "g1",
		CurrentUser: currentUser("u1", shared.RoleContestant),
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Member == nil {
		t.Fatal("expected non-nil member")
	}
	if out.Member.Role() != domainGroup.MemberRoleMember {
		t.Errorf("Role = %v, want MEMBER", out.Member.Role())
	}
	if out.Member.GroupID() != "g1" {
		t.Errorf("GroupID = %q, want %q", out.Member.GroupID(), "g1")
	}
	if out.Member.UserID().Value() != "u1" {
		t.Errorf("UserID = %q, want %q", out.Member.UserID().Value(), "u1")
	}
	if memberRepo.savedMember == nil {
		t.Error("expected Save to be called with the new member")
	}
}

func TestJoinGroup_SaveFailurePropagatesError(t *testing.T) {
	g := openGroup(t)
	memberRepo := &fakeMemberRepo{saveErr: errors.New("db failure")}
	uc := NewJoinGroupUseCase(&fakeRepo{groups: []*domainGroup.Group{g}}, memberRepo)

	_, err := uc.Execute(context.Background(), JoinGroupInput{
		GroupID:     "g1",
		CurrentUser: currentUser("u1", shared.RoleContestant),
	})

	if err == nil {
		t.Fatal("expected error from Save, got nil")
	}
}
