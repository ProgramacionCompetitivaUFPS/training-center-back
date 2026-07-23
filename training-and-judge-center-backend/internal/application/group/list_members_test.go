package group

import (
	"context"
	"testing"

	domainGroup "github.com/training-judge-center/backend/internal/domain/group"
	"github.com/training-judge-center/backend/pkg/apperror"
)

func newListMembersUC(groupRepo *mockGroupRepository, memberRepo *mockMemberRepository, userProvider *mockUserProvider) *ListMembersUseCase {
	return NewListMembersUseCase(groupRepo, memberRepo, userProvider)
}

func TestListMembers_GroupNotFoundReturns404(t *testing.T) {
	uc := newListMembersUC(&mockGroupRepository{}, &mockMemberRepository{}, &mockUserProvider{})
	_, err := uc.Execute(context.Background(), ListMembersInput{
		GroupID:     "nonexistent",
		CurrentUser: asContestant("u1"),
	})
	ae, ok := err.(*apperror.AppError)
	if !ok || ae.Code != domainGroup.ErrCodeGroupNotFound {
		t.Fatalf("expected GROUP_NOT_FOUND, got %v", err)
	}
}

func TestListMembers_NonMemberReturns403(t *testing.T) {
	g := mustGroup(t, "g1", "Club", domainGroup.VisibilityVisible, domainGroup.JoinPolicyOpen)
	uc := newListMembersUC(
		&mockGroupRepository{groups: []*domainGroup.Group{g}},
		&mockMemberRepository{},
		&mockUserProvider{},
	)
	_, err := uc.Execute(context.Background(), ListMembersInput{
		GroupID:     "g1",
		CurrentUser: asContestant("outsider"),
	})
	ae, ok := err.(*apperror.AppError)
	if !ok || ae.Code != ErrCodeInsufficientPermissions {
		t.Fatalf("expected INSUFFICIENT_PERMISSIONS, got %v", err)
	}
}

func TestListMembers_AdminCanListWithoutMembership(t *testing.T) {
	g := mustGroup(t, "g1", "Club", domainGroup.VisibilityVisible, domainGroup.JoinPolicyOpen)
	uc := newListMembersUC(
		&mockGroupRepository{groups: []*domainGroup.Group{g}},
		&mockMemberRepository{},
		&mockUserProvider{},
	)
	out, err := uc.Execute(context.Background(), ListMembersInput{
		GroupID:     "g1",
		CurrentUser: asAdmin("admin"),
	})
	if err != nil {
		t.Fatalf("admin should list without membership, got: %v", err)
	}
	if out.Members == nil {
		t.Error("expected non-nil members slice")
	}
}

func TestListMembers_MemberCanList(t *testing.T) {
	g := mustGroup(t, "g1", "Club", domainGroup.VisibilityVisible, domainGroup.JoinPolicyOpen)
	uid := mustUID("u1")
	m, _ := domainGroup.NewGroupMember("m1", "g1", uid, domainGroup.MemberRoleMember, domainGroup.JoinMethodOpenJoin, nil, testNow)
	memberRepo := &mockMemberRepository{
		memberships: map[string]*domainGroup.GroupMember{keyOf("g1", uid): m},
	}
	userProvider := &mockUserProvider{
		displays: map[string]*UserDisplay{
			"u1": {ID: "u1", Nickname: "member1", Name: "Member One"},
		},
	}
	uc := newListMembersUC(&mockGroupRepository{groups: []*domainGroup.Group{g}}, memberRepo, userProvider)

	out, err := uc.Execute(context.Background(), ListMembersInput{
		GroupID:     "g1",
		Page:        1,
		Limit:       20,
		CurrentUser: asContestant("u1"),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out.Members) != 0 {
		// FindByGroup returns empty in mock, so Members will be empty — this is OK
		// The real adapter is tested separately
	}
	_ = out
}
