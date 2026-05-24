package group

import (
	"context"
	"testing"

	domainGroup "github.com/training-judge-center/backend/internal/domain/group"
	"github.com/training-judge-center/backend/internal/domain/shared"
	"github.com/training-judge-center/backend/pkg/apperror"
)


func requestGroup(t *testing.T) *domainGroup.Group {
	t.Helper()
	return mustGroup(t, "g1", "Algo Club", domainGroup.VisibilityVisible, domainGroup.JoinPolicyRequest)
}

func TestRequestJoin_GroupNotFoundReturns404(t *testing.T) {
	uc := NewRequestJoinUseCase(&mockGroupRepository{}, &mockMemberRepository{}, &mockJoinRequestRepository{})

	_, err := uc.Execute(context.Background(), RequestJoinInput{
		GroupID:     "nonexistent",
		CurrentUser: asContestant("u1"),
	})
	ae, ok := err.(*apperror.AppError)
	if !ok || ae.Code != domainGroup.ErrCodeGroupNotFound {
		t.Fatalf("expected GROUP_NOT_FOUND, got %v", err)
	}
}

func TestRequestJoin_NotVisibleGroupReturns404(t *testing.T) {
	g := mustGroup(t, "g1", "Secret", domainGroup.VisibilityNotVisible, domainGroup.JoinPolicyInvite)
	repo := &mockGroupRepository{groups: []*domainGroup.Group{g}}
	uc := NewRequestJoinUseCase(repo, &mockMemberRepository{}, &mockJoinRequestRepository{})

	_, err := uc.Execute(context.Background(), RequestJoinInput{
		GroupID:     "g1",
		CurrentUser: asContestant("u1"),
	})
	ae, ok := err.(*apperror.AppError)
	if !ok || ae.Code != domainGroup.ErrCodeGroupNotFound {
		t.Fatalf("expected GROUP_NOT_FOUND, got %v", err)
	}
}

func TestRequestJoin_OpenGroupReturns400InvalidPolicy(t *testing.T) {
	g := mustGroup(t, "g1", "Open Club", domainGroup.VisibilityVisible, domainGroup.JoinPolicyOpen)
	repo := &mockGroupRepository{groups: []*domainGroup.Group{g}}
	uc := NewRequestJoinUseCase(repo, &mockMemberRepository{}, &mockJoinRequestRepository{})

	_, err := uc.Execute(context.Background(), RequestJoinInput{
		GroupID:     "g1",
		CurrentUser: asContestant("u1"),
	})
	ae, ok := err.(*apperror.AppError)
	if !ok || ae.Code != ErrCodeInvalidJoinPolicy {
		t.Fatalf("expected INVALID_JOIN_POLICY, got %v", err)
	}
}

func TestRequestJoin_AlreadyMemberReturns409(t *testing.T) {
	g := requestGroup(t)
	userID := shared.RestoreUserID("u1")
	member, _ := domainGroup.NewGroupMember("m1", "g1", userID, domainGroup.MemberRoleMember, testNow)

	repo := &mockGroupRepository{groups: []*domainGroup.Group{g}}
	memberRepo := &mockMemberRepository{memberships: map[string]*domainGroup.GroupMember{keyOf("g1", userID): member}}
	uc := NewRequestJoinUseCase(repo, memberRepo, &mockJoinRequestRepository{})

	_, err := uc.Execute(context.Background(), RequestJoinInput{
		GroupID:     "g1",
		CurrentUser: asContestant("u1"),
	})
	ae, ok := err.(*apperror.AppError)
	if !ok || ae.Code != domainGroup.ErrCodeAlreadyMember {
		t.Fatalf("expected ALREADY_MEMBER, got %v", err)
	}
}

func TestRequestJoin_AlreadyPendingReturns409(t *testing.T) {
	g := requestGroup(t)
	existingReq := mustJoinRequest(t, "r1", "g1", "u1")

	repo := &mockGroupRepository{groups: []*domainGroup.Group{g}}
	reqRepo := &mockJoinRequestRepository{requests: []*domainGroup.JoinRequest{existingReq}}
	uc := NewRequestJoinUseCase(repo, &mockMemberRepository{}, reqRepo)

	_, err := uc.Execute(context.Background(), RequestJoinInput{
		GroupID:     "g1",
		CurrentUser: asContestant("u1"),
	})
	ae, ok := err.(*apperror.AppError)
	if !ok || ae.Code != domainGroup.ErrCodeRequestAlreadyPending {
		t.Fatalf("expected REQUEST_ALREADY_PENDING, got %v", err)
	}
}

func TestRequestJoin_SuccessCreatesRequest(t *testing.T) {
	g := requestGroup(t)
	repo := &mockGroupRepository{groups: []*domainGroup.Group{g}}
	reqRepo := &mockJoinRequestRepository{}
	uc := NewRequestJoinUseCase(repo, &mockMemberRepository{}, reqRepo)

	msg := "I want to join"
	out, err := uc.Execute(context.Background(), RequestJoinInput{
		GroupID:     "g1",
		Message:     &msg,
		CurrentUser: asContestant("u1"),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Request.GroupID != "g1" {
		t.Errorf("expected GroupID=g1, got %s", out.Request.GroupID)
	}
	if out.Request.Status != domainGroup.JoinRequestStatusPending.String() {
		t.Errorf("expected status PENDING, got %s", out.Request.Status)
	}
	if len(reqRepo.savedRequests) != 1 {
		t.Errorf("expected 1 saved request, got %d", len(reqRepo.savedRequests))
	}
}
