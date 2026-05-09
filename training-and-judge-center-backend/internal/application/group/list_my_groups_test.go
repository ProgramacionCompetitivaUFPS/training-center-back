package group

import (
	"context"
	"testing"
	"time"

	domainGroup "github.com/training-judge-center/backend/internal/domain/group"
	"github.com/training-judge-center/backend/internal/domain/shared"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type fakePreferences struct {
	hide bool
}

func (f *fakePreferences) HideGlobalGroup(ctx context.Context, userID string) (bool, error) {
	return f.hide, nil
}

func TestListMyGroups_InvalidRoleReturnsValidationError(t *testing.T) {
	uc := NewListMyGroupsUseCase(&fakeRepo{}, &fakeMemberRepo{}, &fakePreferences{})
	role := "OWNER"

	_, err := uc.Execute(context.Background(), ListMyGroupsInput{
		CurrentUser: currentUser("u1", shared.RoleContestant),
		Role:        &role,
		Page:        1,
		Limit:       20,
	})
	if err == nil {
		t.Fatal("expected validation error for invalid role")
	}
	ae, ok := err.(*apperror.AppError)
	if !ok || ae.Code != apperror.ErrCodeValidationError {
		t.Fatalf("expected VALIDATION_ERROR, got %v", err)
	}
}

func TestListMyGroups_ExcludeDefaultSetWhenPreferenceTrue(t *testing.T) {
	repo := &fakeRepo{}
	uc := NewListMyGroupsUseCase(repo, &fakeMemberRepo{}, &fakePreferences{hide: true})

	_, err := uc.Execute(context.Background(), ListMyGroupsInput{
		CurrentUser: currentUser("u1", shared.RoleContestant),
		Page:        1,
		Limit:       20,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !repo.lastFilters.ExcludeDefault {
		t.Error("expected ExcludeDefault=true when hideGlobalGroup=true")
	}
	if repo.lastFilters.OnlyMyGroups == nil {
		t.Error("expected OnlyMyGroups to be set")
	}
}

func TestListMyGroups_AdminHasNoImplicitMemberships(t *testing.T) {
	repo := &fakeRepo{}
	uc := NewListMyGroupsUseCase(repo, &fakeMemberRepo{}, &fakePreferences{})

	_, err := uc.Execute(context.Background(), ListMyGroupsInput{
		CurrentUser: currentUser("admin", shared.RoleAdmin),
		Page:        1,
		Limit:       20,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.lastFilters.ViewerIsAdmin {
		t.Error("/me/groups must ignore admin implicit permissions (FR-VG-025)")
	}
}

func TestListMyGroups_EnrichesEachResult(t *testing.T) {
	g := mustGroup(t, "g1", "Club", domainGroup.VisibilityVisible, domainGroup.JoinPolicyOpen)
	uid := shared.RestoreUserID("u1")
	gm, _ := domainGroup.NewGroupMember("m1", "g1", uid, domainGroup.MemberRoleLead,
		time.Date(2026, 2, 1, 10, 0, 0, 0, time.UTC))

	repo := &fakeRepo{groups: []*domainGroup.Group{g}, total: 1}
	memberRepo := &fakeMemberRepo{
		memberships:  map[string]*domainGroup.GroupMember{keyOf("g1", uid): gm},
		memberCounts: map[string]int{"g1": 3},
	}
	uc := NewListMyGroupsUseCase(repo, memberRepo, &fakePreferences{})

	out, err := uc.Execute(context.Background(), ListMyGroupsInput{
		CurrentUser: currentUser("u1", shared.RoleContestant),
		Page:        1,
		Limit:       20,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out.Groups) != 1 {
		t.Fatalf("expected 1 group, got %d", len(out.Groups))
	}
	if out.Groups[0].MyRole != domainGroup.MemberRoleLead.String() {
		t.Errorf("expected LEAD, got %v", out.Groups[0].MyRole)
	}
	if out.Groups[0].MemberCount != 3 {
		t.Errorf("expected memberCount=3, got %d", out.Groups[0].MemberCount)
	}
	if out.Groups[0].JoinedAt.IsZero() {
		t.Error("expected joinedAt to be set")
	}
}
