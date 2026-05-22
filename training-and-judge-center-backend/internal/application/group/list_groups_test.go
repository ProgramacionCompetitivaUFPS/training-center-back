package group

import (
	"context"
	"testing"
	"time"

	domainGroup "github.com/training-judge-center/backend/internal/domain/group"
	"github.com/training-judge-center/backend/internal/domain/shared"
	appshared "github.com/training-judge-center/backend/internal/application/shared"
	"github.com/training-judge-center/backend/pkg/apperror"
)

// --- fakes ---

type fakeRepo struct {
	groups         []*domainGroup.Group
	total          int
	lastFilters    domainGroup.ListFilters
	returnErr      error
	existsByNameFn func(name domainGroup.GroupName) (bool, error)
	saveErr        error
}

func (f *fakeRepo) Save(ctx context.Context, g *domainGroup.Group) error { return f.saveErr }
func (f *fakeRepo) FindByID(ctx context.Context, id string) (*domainGroup.Group, error) {
	for _, g := range f.groups {
		if g.ID() == id {
			return g, nil
		}
	}
	return nil, apperror.NewNotFound(domainGroup.ErrCodeGroupNotFound, "group not found")
}
func (f *fakeRepo) ExistsByName(ctx context.Context, n domainGroup.GroupName) (bool, error) {
	if f.existsByNameFn != nil {
		return f.existsByNameFn(n)
	}
	return false, nil
}
func (f *fakeRepo) FindDefault(ctx context.Context) (*domainGroup.Group, error) { return nil, nil }
func (f *fakeRepo) Delete(ctx context.Context, id string) error                 { return nil }
func (f *fakeRepo) List(ctx context.Context, filters domainGroup.ListFilters) ([]*domainGroup.Group, int, error) {
	f.lastFilters = filters
	if f.returnErr != nil {
		return nil, 0, f.returnErr
	}
	return f.groups, f.total, nil
}

type fakeMemberRepo struct {
	memberCounts         map[string]int
	memberships          map[string]*domainGroup.GroupMember // key: groupID+userID
	leadCounts           map[string]int
	leads                map[string][]*domainGroup.GroupMember
	saveErr              error
	savedMember          *domainGroup.GroupMember
	findByGroupAndUserErr error
}

func keyOf(groupID string, userID shared.UserID) string { return groupID + "::" + userID.Value() }

func (f *fakeMemberRepo) Save(_ context.Context, m *domainGroup.GroupMember) error {
	f.savedMember = m
	return f.saveErr
}
func (f *fakeMemberRepo) SaveAll(ctx context.Context, members []*domainGroup.GroupMember) error {
	return nil
}
func (f *fakeMemberRepo) FindByGroupAndUser(ctx context.Context, groupID string, userID shared.UserID) (*domainGroup.GroupMember, error) {
	if f.findByGroupAndUserErr != nil {
		return nil, f.findByGroupAndUserErr
	}
	if m, ok := f.memberships[keyOf(groupID, userID)]; ok {
		return m, nil
	}
	return nil, nil
}
func (f *fakeMemberRepo) FindByGroup(ctx context.Context, groupID string, filters domainGroup.MemberFilters) ([]*domainGroup.GroupMember, int, error) {
	return nil, 0, nil
}
func (f *fakeMemberRepo) Delete(ctx context.Context, groupID string, userID shared.UserID) error {
	return nil
}
func (f *fakeMemberRepo) CountLeads(ctx context.Context, groupID string) (int, error) {
	return f.leadCounts[groupID], nil
}
func (f *fakeMemberRepo) CountMembers(ctx context.Context, groupID string) (int, error) {
	return f.memberCounts[groupID], nil
}
func (f *fakeMemberRepo) ListLeads(ctx context.Context, groupID string) ([]*domainGroup.GroupMember, error) {
	return f.leads[groupID], nil
}
func (f *fakeMemberRepo) BulkStats(ctx context.Context, groupIDs []string, viewerID shared.UserID) (map[string]domainGroup.MemberStats, error) {
	result := make(map[string]domainGroup.MemberStats, len(groupIDs))
	for _, gid := range groupIDs {
		s := domainGroup.MemberStats{Count: f.memberCounts[gid]}
		if m := f.memberships[keyOf(gid, viewerID)]; m != nil {
			s.IsMember = true
			s.Role = m.Role()
			s.JoinedAt = m.JoinedAt()
		}
		result[gid] = s
	}
	return result, nil
}

// --- helpers ---

var testNow = time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)

func mustGroup(t *testing.T, id, name string, visibility domainGroup.Visibility, joinPolicy domainGroup.JoinPolicy) *domainGroup.Group {
	t.Helper()
	gn, err := domainGroup.NewGroupName(name)
	if err != nil {
		t.Fatalf("NewGroupName: %v", err)
	}
	g, err := domainGroup.NewGroup(id, gn, nil, visibility, joinPolicy, shared.RestoreUserID("creator-id"),
		time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("NewGroup: %v", err)
	}
	return g
}

func currentUser(id string, role shared.Role) appshared.CurrentUser {
	return appshared.CurrentUser{ID: id, Role: role}
}

type fakeJoinRequestRepo struct {
	requests      []*domainGroup.JoinRequest
	savedRequests []*domainGroup.JoinRequest
	deletedIDs    []string
	saveErr       error
	findErr       error
}

func (f *fakeJoinRequestRepo) Save(_ context.Context, r *domainGroup.JoinRequest) error {
	if f.saveErr != nil {
		return f.saveErr
	}
	for i, existing := range f.savedRequests {
		if existing.ID() == r.ID() {
			f.savedRequests[i] = r
			return nil
		}
	}
	f.savedRequests = append(f.savedRequests, r)
	return nil
}

func (f *fakeJoinRequestRepo) FindByID(_ context.Context, id string) (*domainGroup.JoinRequest, error) {
	if f.findErr != nil {
		return nil, f.findErr
	}
	for _, r := range f.requests {
		if r.ID() == id {
			return r, nil
		}
	}
	return nil, nil
}

func (f *fakeJoinRequestRepo) FindByGroupAndUser(_ context.Context, groupID string, userID shared.UserID) (*domainGroup.JoinRequest, error) {
	if f.findErr != nil {
		return nil, f.findErr
	}
	var latest *domainGroup.JoinRequest
	for _, r := range f.requests {
		if r.GroupID() == groupID && r.RequesterUserID().Value() == userID.Value() {
			if r.IsPending() {
				return r, nil
			}
			if latest == nil {
				latest = r
			}
		}
	}
	return latest, nil
}

func (f *fakeJoinRequestRepo) FindByGroup(_ context.Context, groupID string, filters domainGroup.JoinRequestFilters) ([]*domainGroup.JoinRequest, int, error) {
	var out []*domainGroup.JoinRequest
	for _, r := range f.requests {
		if r.GroupID() != groupID {
			continue
		}
		if filters.Status != nil && r.Status() != *filters.Status {
			continue
		}
		out = append(out, r)
	}
	return out, len(out), nil
}

func (f *fakeJoinRequestRepo) Delete(_ context.Context, id string) error {
	f.deletedIDs = append(f.deletedIDs, id)
	return nil
}

type fakeTxManager struct{}

func (f *fakeTxManager) WithTx(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}

func mustJoinRequest(t *testing.T, id, groupID, userID string) *domainGroup.JoinRequest {
	t.Helper()
	req, err := domainGroup.NewJoinRequest(
		id, groupID, shared.RestoreUserID(userID), nil,
		time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatalf("NewJoinRequest: %v", err)
	}
	return req
}

// --- tests ---

func TestListGroups_PageValidation(t *testing.T) {
	uc := NewListGroupsUseCase(&fakeRepo{}, &fakeMemberRepo{})
	_, err := uc.Execute(context.Background(), ListGroupsInput{
		CurrentUser: currentUser("u1", shared.RoleContestant),
		Page:        0,
		Limit:       20,
	})
	if err == nil {
		t.Fatal("expected validation error for page<1")
	}
}

func TestListGroups_LimitValidation(t *testing.T) {
	uc := NewListGroupsUseCase(&fakeRepo{}, &fakeMemberRepo{})
	_, err := uc.Execute(context.Background(), ListGroupsInput{
		CurrentUser: currentUser("u1", shared.RoleContestant),
		Page:        1,
		Limit:       100,
	})
	if err == nil {
		t.Fatal("expected validation error for limit>50")
	}
}

func TestListGroups_PassesViewerFlagsToRepo(t *testing.T) {
	repo := &fakeRepo{}
	uc := NewListGroupsUseCase(repo, &fakeMemberRepo{memberCounts: map[string]int{}})

	_, err := uc.Execute(context.Background(), ListGroupsInput{
		CurrentUser: currentUser("admin-id", shared.RoleAdmin),
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

	repo := &fakeRepo{groups: []*domainGroup.Group{g}, total: 1}
	memberRepo := &fakeMemberRepo{
		memberCounts: map[string]int{"g1": 10},
		memberships:  map[string]*domainGroup.GroupMember{keyOf("g1", userID): gm},
	}
	uc := NewListGroupsUseCase(repo, memberRepo)

	out, err := uc.Execute(context.Background(), ListGroupsInput{
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
	if out.Groups[0].MemberCount != 10 {
		t.Errorf("expected memberCount=10, got %d", out.Groups[0].MemberCount)
	}
	if out.Groups[0].UserRole != domainGroup.MemberRoleLead.String() {
		t.Errorf("expected role=LEAD, got %v", out.Groups[0].UserRole)
	}
}

func TestListGroups_InvalidSortBy(t *testing.T) {
	uc := NewListGroupsUseCase(&fakeRepo{}, &fakeMemberRepo{})
	_, err := uc.Execute(context.Background(), ListGroupsInput{
		CurrentUser: currentUser("u1", shared.RoleContestant),
		SortBy:      "nope",
		Page:        1,
		Limit:       20,
	})
	if err == nil {
		t.Fatal("expected error for invalid sortBy")
	}
}
