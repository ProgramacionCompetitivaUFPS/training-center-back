package group

import (
	"context"
	"testing"

	domainGroup "github.com/training-judge-center/backend/internal/domain/group"
	"github.com/training-judge-center/backend/pkg/apperror"
)

func newLeaveGroupUC(groupRepo *mockGroupRepository, memberRepo *mockMemberRepository) *LeaveGroupUseCase {
	return NewLeaveGroupUseCase(groupRepo, memberRepo)
}

func TestLeaveGroup_GlobalGroupReturns400(t *testing.T) {
	g := domainGroup.RestoreGroup("g-global", domainGroup.RestoreGroupName("Global"),
		nil, domainGroup.VisibilityVisible, domainGroup.JoinPolicyOpen, true,
		mustUID("creator"), testNow, testNow)
	uc := newLeaveGroupUC(&mockGroupRepository{groups: []*domainGroup.Group{g}}, &mockMemberRepository{})

	err := uc.Execute(context.Background(), LeaveGroupInput{
		GroupID:     "g-global",
		CurrentUser: asContestant("u1"),
	})
	ae, ok := err.(*apperror.AppError)
	if !ok || ae.Code != domainGroup.ErrCodeCannotLeaveGlobalGroup {
		t.Fatalf("expected CANNOT_LEAVE_GLOBAL_GROUP, got %v", err)
	}
}

func TestLeaveGroup_NotAMemberReturns404(t *testing.T) {
	g := mustGroup(t, "g1", "Club", domainGroup.VisibilityVisible, domainGroup.JoinPolicyOpen)
	uc := newLeaveGroupUC(&mockGroupRepository{groups: []*domainGroup.Group{g}}, &mockMemberRepository{})

	err := uc.Execute(context.Background(), LeaveGroupInput{
		GroupID:     "g1",
		CurrentUser: asContestant("u1"),
	})
	ae, ok := err.(*apperror.AppError)
	if !ok || ae.Code != domainGroup.ErrCodeNotAMember {
		t.Fatalf("expected NOT_A_MEMBER, got %v", err)
	}
}

func TestLeaveGroup_LastLeadReturns400(t *testing.T) {
	g := mustGroup(t, "g1", "Club", domainGroup.VisibilityVisible, domainGroup.JoinPolicyOpen)
	memberRepo := leadMemberRepo("g1", "lead")
	memberRepo.leadCounts = map[string]int{"g1": 1}
	uc := newLeaveGroupUC(&mockGroupRepository{groups: []*domainGroup.Group{g}}, memberRepo)

	err := uc.Execute(context.Background(), LeaveGroupInput{
		GroupID:     "g1",
		CurrentUser: asCoach("lead"),
	})
	ae, ok := err.(*apperror.AppError)
	if !ok || ae.Code != domainGroup.ErrCodeCannotLeaveAsLastLead {
		t.Fatalf("expected CANNOT_LEAVE_AS_LAST_LEAD, got %v", err)
	}
}

func TestLeaveGroup_RegularMemberSucceeds(t *testing.T) {
	g := mustGroup(t, "g1", "Club", domainGroup.VisibilityVisible, domainGroup.JoinPolicyOpen)
	uid := mustUID("u1")
	m, _ := domainGroup.NewGroupMember("m1", "g1", uid, domainGroup.MemberRoleMember, domainGroup.JoinMethodOpenJoin, nil, testNow)
	memberRepo := &mockMemberRepository{
		memberships: map[string]*domainGroup.GroupMember{keyOf("g1", uid): m},
	}
	uc := newLeaveGroupUC(&mockGroupRepository{groups: []*domainGroup.Group{g}}, memberRepo)

	err := uc.Execute(context.Background(), LeaveGroupInput{
		GroupID:     "g1",
		CurrentUser: asContestant("u1"),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLeaveGroup_LeadWithOtherLeadsSucceeds(t *testing.T) {
	g := mustGroup(t, "g1", "Club", domainGroup.VisibilityVisible, domainGroup.JoinPolicyOpen)
	memberRepo := leadMemberRepo("g1", "lead")
	memberRepo.leadCounts = map[string]int{"g1": 2}
	uc := newLeaveGroupUC(&mockGroupRepository{groups: []*domainGroup.Group{g}}, memberRepo)

	err := uc.Execute(context.Background(), LeaveGroupInput{
		GroupID:     "g1",
		CurrentUser: asCoach("lead"),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
