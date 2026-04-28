package material

import (
	"context"
	"time"

	domainMaterial "github.com/training-judge-center/backend/internal/domain/material"
	"github.com/training-judge-center/backend/internal/domain/shared"
	"github.com/training-judge-center/backend/pkg/apperror"
)

var testNow = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

func fixedClock() time.Time { return testNow }

// ── Repository mock ──────────────────────────────────────────────────────────

type mockMaterialRepository struct {
	saveFn     func(ctx context.Context, m *domainMaterial.Material) error
	findByIDFn func(ctx context.Context, id string) (*domainMaterial.Material, error)
	listFn     func(ctx context.Context, groupID string, f domainMaterial.ListFilters) ([]*domainMaterial.Material, int, error)
	deleteFn   func(ctx context.Context, id string) error
}

func (m *mockMaterialRepository) Save(ctx context.Context, mat *domainMaterial.Material) error {
	if m.saveFn != nil {
		return m.saveFn(ctx, mat)
	}
	return nil
}
func (m *mockMaterialRepository) FindByID(ctx context.Context, id string) (*domainMaterial.Material, error) {
	if m.findByIDFn != nil {
		return m.findByIDFn(ctx, id)
	}
	return nil, apperror.NewNotFound(domainMaterial.ErrCodeMaterialNotFound, "material not found")
}
func (m *mockMaterialRepository) List(ctx context.Context, groupID string, f domainMaterial.ListFilters) ([]*domainMaterial.Material, int, error) {
	if m.listFn != nil {
		return m.listFn(ctx, groupID, f)
	}
	return nil, 0, nil
}
func (m *mockMaterialRepository) Delete(ctx context.Context, id string) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, id)
	}
	return nil
}

// ── GroupProvider mock ───────────────────────────────────────────────────────

type mockGroupProvider struct {
	existsFn func(ctx context.Context, groupID string) (bool, error)
}

func (m *mockGroupProvider) Exists(ctx context.Context, groupID string) (bool, error) {
	if m.existsFn != nil {
		return m.existsFn(ctx, groupID)
	}
	return true, nil
}

// ── GroupVisibilityProvider mock ─────────────────────────────────────────────

type mockGroupVisibilityProvider struct {
	findVisibilityFn func(ctx context.Context, groupID string) (GroupVisibility, bool, error)
}

func (m *mockGroupVisibilityProvider) FindVisibility(ctx context.Context, groupID string) (GroupVisibility, bool, error) {
	if m.findVisibilityFn != nil {
		return m.findVisibilityFn(ctx, groupID)
	}
	return GroupVisibilityVisible, true, nil
}

func visibleGroup() *mockGroupVisibilityProvider {
	return &mockGroupVisibilityProvider{}
}

func notVisibleGroup() *mockGroupVisibilityProvider {
	return &mockGroupVisibilityProvider{
		findVisibilityFn: func(_ context.Context, _ string) (GroupVisibility, bool, error) {
			return GroupVisibilityNotVisible, true, nil
		},
	}
}

func groupVisibilityNotFound() *mockGroupVisibilityProvider {
	return &mockGroupVisibilityProvider{
		findVisibilityFn: func(_ context.Context, _ string) (GroupVisibility, bool, error) {
			return "", false, nil
		},
	}
}

// ── GroupMemberProvider mock ─────────────────────────────────────────────────

type mockGroupMemberProvider struct {
	isLeadFn    func(ctx context.Context, userID, groupID string) (bool, error)
	isMemberFn  func(ctx context.Context, userID, groupID string) (bool, error)
}

func (m *mockGroupMemberProvider) IsLeadOfGroup(ctx context.Context, userID, groupID string) (bool, error) {
	if m.isLeadFn != nil {
		return m.isLeadFn(ctx, userID, groupID)
	}
	return false, nil
}

func (m *mockGroupMemberProvider) IsMemberOfGroup(ctx context.Context, userID, groupID string) (bool, error) {
	if m.isMemberFn != nil {
		return m.isMemberFn(ctx, userID, groupID)
	}
	return false, nil
}

func isLead() *mockGroupMemberProvider {
	return &mockGroupMemberProvider{
		isLeadFn:   func(_ context.Context, _, _ string) (bool, error) { return true, nil },
		isMemberFn: func(_ context.Context, _, _ string) (bool, error) { return true, nil },
	}
}

func notLead() *mockGroupMemberProvider {
	return &mockGroupMemberProvider{
		isLeadFn:   func(_ context.Context, _, _ string) (bool, error) { return false, nil },
		isMemberFn: func(_ context.Context, _, _ string) (bool, error) { return false, nil },
	}
}

func isMemberNotLead() *mockGroupMemberProvider {
	return &mockGroupMemberProvider{
		isLeadFn:   func(_ context.Context, _, _ string) (bool, error) { return false, nil },
		isMemberFn: func(_ context.Context, _, _ string) (bool, error) { return true, nil },
	}
}

// ── AuthorProvider mock ──────────────────────────────────────────────────────

type mockAuthorProvider struct {
	getDisplaysFn func(ctx context.Context, userIDs []string) (map[string]*AuthorDisplay, error)
}

func (m *mockAuthorProvider) GetDisplays(ctx context.Context, userIDs []string) (map[string]*AuthorDisplay, error) {
	if m.getDisplaysFn != nil {
		return m.getDisplaysFn(ctx, userIDs)
	}
	result := make(map[string]*AuthorDisplay, len(userIDs))
	for _, id := range userIDs {
		result[id] = &AuthorDisplay{Nickname: "author", Name: "Author Name"}
	}
	return result, nil
}

func stubAuthorProvider() *mockAuthorProvider { return &mockAuthorProvider{} }

// ── CurrentUser helpers ──────────────────────────────────────────────────────

func asAdmin(id string) shared.CurrentUser      { return shared.CurrentUser{ID: id, Role: shared.RoleAdmin} }
func asCoach(id string) shared.CurrentUser      { return shared.CurrentUser{ID: id, Role: shared.RoleCoach} }
func asContestant(id string) shared.CurrentUser { return shared.CurrentUser{ID: id, Role: shared.RoleContestant} }

// ── Material fixtures ────────────────────────────────────────────────────────

const (
	testAuthorID   = "aaaaaaaa-0000-0000-0000-000000000001"
	testOtherID    = "aaaaaaaa-0000-0000-0000-000000000002"
	testGroupID    = "bbbbbbbb-0000-0000-0000-000000000001"
	testMaterialID = "cccccccc-0000-0000-0000-000000000001"
)

func newTestMaterial() *domainMaterial.Material {
	return domainMaterial.RestoreMaterial(
		testMaterialID,
		testGroupID,
		shared.RestoreUserID(testAuthorID),
		"Test Title",
		"",
		nil,
		"DRAFT",
		false,
		nil,
		testNow,
		testNow,
		nil,
	).WithClock(fixedClock)
}

func newPublishedMaterial() *domainMaterial.Material {
	now := testNow
	return domainMaterial.RestoreMaterial(
		testMaterialID,
		testGroupID,
		shared.RestoreUserID(testAuthorID),
		"Test Title",
		"",
		nil,
		"PUBLISHED",
		false,
		nil,
		testNow,
		testNow,
		&now,
	).WithClock(fixedClock)
}

func repoWith(m *domainMaterial.Material) *mockMaterialRepository {
	return &mockMaterialRepository{
		findByIDFn: func(_ context.Context, _ string) (*domainMaterial.Material, error) {
			return m, nil
		},
	}
}

func repoWithList(materials []*domainMaterial.Material) *mockMaterialRepository {
	return &mockMaterialRepository{
		listFn: func(_ context.Context, _ string, _ domainMaterial.ListFilters) ([]*domainMaterial.Material, int, error) {
			return materials, len(materials), nil
		},
	}
}

func groupExists() *mockGroupProvider {
	return &mockGroupProvider{existsFn: func(_ context.Context, _ string) (bool, error) { return true, nil }}
}

func groupNotFound() *mockGroupProvider {
	return &mockGroupProvider{existsFn: func(_ context.Context, _ string) (bool, error) { return false, nil }}
}
