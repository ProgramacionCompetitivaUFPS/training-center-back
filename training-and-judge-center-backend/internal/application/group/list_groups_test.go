package group

import (
	"context"
	"testing"

	domainGroup "github.com/training-judge-center/backend/internal/domain/group"
	"github.com/training-judge-center/backend/internal/domain/shared"
)

func newListGroupsUseCase(repo *mockGroupRepository, memberRepo *mockMemberRepository) *ListGroupsUseCase {
	return NewListGroupsUseCase(repo, memberRepo)
}

func TestListGroups_PageValidation(t *testing.T) {
	uc := NewListGroupsUseCase(&mockGroupRepository{}, &mockMemberRepository{})
	_, err := uc.Execute(context.Background(), ListGroupsInput{
		CurrentUser: asContestant("u1"),
		Page:        0,
		Limit:       20,
	})
	if err == nil {
		t.Fatal("expected validation error for page<1")
	}
}

func TestListGroups_LimitValidation(t *testing.T) {
	uc := NewListGroupsUseCase(&mockGroupRepository{}, &mockMemberRepository{})
	_, err := uc.Execute(context.Background(), ListGroupsInput{
		CurrentUser: asContestant("u1"),
		Page:        1,
		Limit:       100,
	})
	if err == nil {
		t.Fatal("expected validation error for limit>50")
	}
}

func TestListGroups_PassesViewerFlagsToRepo(t *testing.T) {
	repo := &mockGroupRepository{}
	uc := NewListGroupsUseCase(repo, &mockMemberRepository{memberCounts: map[string]int{}})

	_, err := uc.Execute(context.Background(), ListGroupsInput{
		CurrentUser: asAdmin("admin-id"),
		Page:        1,
		Limit:       20,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !repo.lastFilters.ViewerIsAdmin {
		t.Error("expected ViewerIsAdmin=true for admin caller")
	}
	if repo.lastFilters.ViewerID.Value() != "admin-id" {
		t.Errorf("expected ViewerID=admin-id, got %q", repo.lastFilters.ViewerID.Value())
	}
}

func TestListGroups_EnrichesWithMemberCountAndRole(t *testing.T) {
	g := mustGroup(t, "g1", "Club Programming", domainGroup.VisibilityVisible, domainGroup.JoinPolicyOpen)

	userID := shared.RestoreUserID("u1")
	gm, _ := domainGroup.NewGroupMember("m1", "g1", userID, domainGroup.MemberRoleLead, testNow)

	repo := &mockGroupRepository{groups: []*domainGroup.Group{g}, total: 1}
	memberRepo := &mockMemberRepository{
		memberCounts: map[string]int{"g1": 10},
		memberships:  map[string]*domainGroup.GroupMember{keyOf("g1", userID): gm},
	}
	uc := NewListGroupsUseCase(repo, memberRepo)

	out, err := uc.Execute(context.Background(), ListGroupsInput{
		CurrentUser: asContestant("u1"),
		Page:        1,
		Limit:       20,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out.Groups) != 1 {
		t.Fatalf("expected 1 group, got %d", len(out.Groups))
	}
	if out.Groups[0].MemberCount != 10 {
		t.Errorf("expected memberCount=10, got %d", out.Groups[0].MemberCount)
	}
	if out.Groups[0].UserRole != domainGroup.MemberRoleLead.String() {
		t.Errorf("expected role=LEAD, got %v", out.Groups[0].UserRole)
	}
}

func TestListGroups_InvalidSortBy(t *testing.T) {
	uc := NewListGroupsUseCase(&mockGroupRepository{}, &mockMemberRepository{})
	_, err := uc.Execute(context.Background(), ListGroupsInput{
		CurrentUser: asContestant("u1"),
		SortBy:      "nope",
		Page:        1,
		Limit:       20,
	})
	if err == nil {
		t.Fatal("expected error for invalid sortBy")
	}
}
