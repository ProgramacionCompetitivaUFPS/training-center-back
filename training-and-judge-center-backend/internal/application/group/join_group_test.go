package group

import (
	"context"
	"errors"
	"testing"

	domainGroup "github.com/training-judge-center/backend/internal/domain/group"
	"github.com/training-judge-center/backend/internal/domain/shared"
	"github.com/training-judge-center/backend/pkg/apperror"
)

func newJoinGroupUseCase(repo *mockGroupRepository, memberRepo *mockMemberRepository) *JoinGroupUseCase {
	return NewJoinGroupUseCase(repo, memberRepo)
}

func openGroup(t *testing.T) *domainGroup.Group {
	t.Helper()
	return mustGroup(t, "g1", "Open Club", domainGroup.VisibilityVisible, domainGroup.JoinPolicyOpen)
}

func TestJoinGroup_EmptyGroupIDReturnsValidationError(t *testing.T) {
	uc := NewJoinGroupUseCase(&mockGroupRepository{}, &mockMemberRepository{})

	_, err := uc.Execute(context.Background(), JoinGroupInput{
		GroupID:     "",
		CurrentUser: asContestant("u1"),
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
	uc := NewJoinGroupUseCase(&mockGroupRepository{}, &mockMemberRepository{})

	_, err := uc.Execute(context.Background(), JoinGroupInput{
		GroupID:     "nonexistent",
		CurrentUser: asContestant("u1"),
	})

	ae, ok := err.(*apperror.AppError)
	if !ok || ae.Code != domainGroup.ErrCodeGroupNotFound {
		t.Fatalf("expected GROUP_NOT_FOUND, got %v", err)
	}
}

func TestJoinGroup_InvitePolicyReturns403(t *testing.T) {
	g := mustGroup(t, "g1", "Invite Club", domainGroup.VisibilityVisible, domainGroup.JoinPolicyInvite)
	uc := NewJoinGroupUseCase(&mockGroupRepository{groups: []*domainGroup.Group{g}}, &mockMemberRepository{})

	_, err := uc.Execute(context.Background(), JoinGroupInput{
		GroupID:     "g1",
		CurrentUser: asContestant("u1"),
	})

	ae, ok := err.(*apperror.AppError)
	if !ok || ae.Code != ErrCodeInsufficientPermissions {
		t.Fatalf("expected INSUFFICIENT_PERMISSIONS, got %v", err)
	}
	if ae.Kind != apperror.KindForbidden {
		t.Errorf("expected kind FORBIDDEN, got %s", ae.Kind)
	}
}

func TestJoinGroup_RequestPolicyReturns403(t *testing.T) {
	g := mustGroup(t, "g1", "Request Club", domainGroup.VisibilityVisible, domainGroup.JoinPolicyRequest)
	uc := NewJoinGroupUseCase(&mockGroupRepository{groups: []*domainGroup.Group{g}}, &mockMemberRepository{})

	_, err := uc.Execute(context.Background(), JoinGroupInput{
		GroupID:     "g1",
		CurrentUser: asContestant("u1"),
	})

	ae, ok := err.(*apperror.AppError)
	if !ok || ae.Code != ErrCodeInsufficientPermissions {
		t.Fatalf("expected INSUFFICIENT_PERMISSIONS, got %v", err)
	}
}

func TestJoinGroup_AlreadyMemberReturns409(t *testing.T) {
	g := openGroup(t)
	userID := shared.RestoreUserID("u1")
	existingMember := domainGroup.RestoreGroupMember("m1", "g1", userID, domainGroup.MemberRoleMember, testNow)
	memberRepo := &mockMemberRepository{
		memberships: map[string]*domainGroup.GroupMember{
			keyOf("g1", userID): existingMember,
		},
	}
	uc := NewJoinGroupUseCase(&mockGroupRepository{groups: []*domainGroup.Group{g}}, memberRepo)

	_, err := uc.Execute(context.Background(), JoinGroupInput{
		GroupID:     "g1",
		CurrentUser: asContestant("u1"),
	})

	ae, ok := err.(*apperror.AppError)
	if !ok || ae.Code != domainGroup.ErrCodeAlreadyMember {
		t.Fatalf("expected ALREADY_MEMBER, got %v", err)
	}
	if ae.Kind != apperror.KindConflict {
		t.Errorf("expected kind CONFLICT, got %s", ae.Kind)
	}
}

func TestJoinGroup_SuccessSavesMemberWithRoleMember(t *testing.T) {
	g := openGroup(t)
	memberRepo := &mockMemberRepository{}
	uc := NewJoinGroupUseCase(&mockGroupRepository{groups: []*domainGroup.Group{g}}, memberRepo)

	out, err := uc.Execute(context.Background(), JoinGroupInput{
		GroupID:     "g1",
		CurrentUser: asContestant("u1"),
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Member.Role != domainGroup.MemberRoleMember.String() {
		t.Errorf("Role = %v, want MEMBER", out.Member.Role)
	}
	if memberRepo.savedMember == nil {
		t.Error("expected Save to be called with the new member")
	}
}

func TestJoinGroup_SaveFailurePropagatesError(t *testing.T) {
	g := openGroup(t)
	memberRepo := &mockMemberRepository{saveErr: errors.New("db failure")}
	uc := NewJoinGroupUseCase(&mockGroupRepository{groups: []*domainGroup.Group{g}}, memberRepo)

	_, err := uc.Execute(context.Background(), JoinGroupInput{
		GroupID:     "g1",
		CurrentUser: asContestant("u1"),
	})

	if err == nil {
		t.Fatal("expected error from Save, got nil")
	}
}

func TestJoinGroup_FindMemberErrorPropagates(t *testing.T) {
	g := openGroup(t)
	memberRepo := &mockMemberRepository{findByGroupAndUserErr: errors.New("db timeout")}
	uc := NewJoinGroupUseCase(&mockGroupRepository{groups: []*domainGroup.Group{g}}, memberRepo)

	_, err := uc.Execute(context.Background(), JoinGroupInput{
		GroupID:     "g1",
		CurrentUser: asContestant("u1"),
	})

	if err == nil {
		t.Fatal("expected error from FindByGroupAndUser, got nil")
	}
}
