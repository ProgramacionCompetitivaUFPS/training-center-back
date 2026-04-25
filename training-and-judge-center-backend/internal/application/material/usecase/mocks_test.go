package usecase

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

// ── GroupMemberProvider mock ─────────────────────────────────────────────────

type mockGroupMemberProvider struct {
	isLeadFn func(ctx context.Context, userID, groupID string) (bool, error)
}

func (m *mockGroupMemberProvider) IsLeadOfGroup(ctx context.Context, userID, groupID string) (bool, error) {
	if m.isLeadFn != nil {
		return m.isLeadFn(ctx, userID, groupID)
	}
	return false, nil
}

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

func repoWith(m *domainMaterial.Material) *mockMaterialRepository {
	return &mockMaterialRepository{
		findByIDFn: func(_ context.Context, _ string) (*domainMaterial.Material, error) {
			return m, nil
		},
	}
}

func groupExists() *mockGroupProvider {
	return &mockGroupProvider{existsFn: func(_ context.Context, _ string) (bool, error) { return true, nil }}
}

func groupNotFound() *mockGroupProvider {
	return &mockGroupProvider{existsFn: func(_ context.Context, _ string) (bool, error) { return false, nil }}
}

func isLead() *mockGroupMemberProvider {
	return &mockGroupMemberProvider{isLeadFn: func(_ context.Context, _, _ string) (bool, error) { return true, nil }}
}

func notLead() *mockGroupMemberProvider {
	return &mockGroupMemberProvider{isLeadFn: func(_ context.Context, _, _ string) (bool, error) { return false, nil }}
}
