package material

import (
	"context"
	"errors"
	"testing"

	domainMaterial "github.com/training-judge-center/backend/internal/domain/material"
	"github.com/training-judge-center/backend/pkg/apperror"
)

func newUnpinMaterialUseCase(repo *mockMaterialRepository, group *mockGroupProvider, member *mockGroupMemberProvider) *UnpinMaterialUseCase {
	return NewUnpinMaterialUseCase(repo, group, member, stubAuthorProvider())
}

func TestUnpinMaterial_SuccessByAuthor(t *testing.T) {
	m := newPinnedMaterial()
	saved := false
	repo := &mockMaterialRepository{
		findByIDFn: func(_ context.Context, _ string) (*domainMaterial.Material, error) { return m, nil },
		saveFn:     func(_ context.Context, _ *domainMaterial.Material) error { saved = true; return nil },
	}
	uc := newUnpinMaterialUseCase(repo, groupExists(), notLead())

	out, err := uc.Execute(context.Background(), UnpinMaterialInput{
		CurrentUser: asCoach(testAuthorID),
		GroupID:     testGroupID,
		MaterialID:  testMaterialID,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Material.Pinned {
		t.Error("expected pinned=false after unpin")
	}
	if out.Material.PinnedAt != nil {
		t.Error("expected pinnedAt=nil after unpin")
	}
	if !saved {
		t.Error("expected repo.Save to be called")
	}
}

func TestUnpinMaterial_SuccessByOtherLead(t *testing.T) {
	m := newPinnedMaterial()
	uc := newUnpinMaterialUseCase(repoWith(m), groupExists(), isLead())

	out, err := uc.Execute(context.Background(), UnpinMaterialInput{
		CurrentUser: asCoach(testOtherID),
		GroupID:     testGroupID,
		MaterialID:  testMaterialID,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Material.Pinned {
		t.Error("expected pinned=false")
	}
}

func TestUnpinMaterial_SuccessByAdmin(t *testing.T) {
	m := newPinnedMaterial()
	uc := newUnpinMaterialUseCase(repoWith(m), groupExists(), notLead())

	out, err := uc.Execute(context.Background(), UnpinMaterialInput{
		CurrentUser: asAdmin(testOtherID),
		GroupID:     testGroupID,
		MaterialID:  testMaterialID,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Material.Pinned {
		t.Error("expected pinned=false after admin unpin")
	}
}

func TestUnpinMaterial_Admin_SkipsLeadCheck(t *testing.T) {
	m := newPinnedMaterial()
	leadCheckCalled := false
	member := &mockGroupMemberProvider{
		isLeadFn: func(_ context.Context, _, _ string) (bool, error) {
			leadCheckCalled = true
			return false, errors.New("should not be called")
		},
	}
	uc := newUnpinMaterialUseCase(repoWith(m), groupExists(), member)

	_, err := uc.Execute(context.Background(), UnpinMaterialInput{
		CurrentUser: asAdmin(testOtherID),
		GroupID:     testGroupID,
		MaterialID:  testMaterialID,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if leadCheckCalled {
		t.Error("IsLeadOfGroup must not be called for admin users")
	}
}

func TestUnpinMaterial_Idempotent_AlreadyUnpinned(t *testing.T) {
	m := newPublishedMaterial()
	uc := newUnpinMaterialUseCase(repoWith(m), groupExists(), notLead())

	out, err := uc.Execute(context.Background(), UnpinMaterialInput{
		CurrentUser: asCoach(testAuthorID),
		GroupID:     testGroupID,
		MaterialID:  testMaterialID,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Material.Pinned {
		t.Error("expected pinned=false")
	}
	if out.Material.PinnedAt != nil {
		t.Error("expected pinnedAt=nil")
	}
}

func TestUnpinMaterial_Forbidden_Member(t *testing.T) {
	m := newPinnedMaterial()
	uc := newUnpinMaterialUseCase(repoWith(m), groupExists(), isMemberNotLead())

	_, err := uc.Execute(context.Background(), UnpinMaterialInput{
		CurrentUser: asCoach(testOtherID),
		GroupID:     testGroupID,
		MaterialID:  testMaterialID,
	})

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Code != ErrCodeInsufficientPermissions {
		t.Errorf("expected INSUFFICIENT_PERMISSIONS, got %v", err)
	}
}

func TestUnpinMaterial_MemberProviderError_Returns500(t *testing.T) {
	m := newPinnedMaterial()
	member := &mockGroupMemberProvider{
		isLeadFn: func(_ context.Context, _, _ string) (bool, error) {
			return false, apperror.NewInternal()
		},
	}
	uc := newUnpinMaterialUseCase(repoWith(m), groupExists(), member)

	_, err := uc.Execute(context.Background(), UnpinMaterialInput{
		CurrentUser: asCoach(testOtherID),
		GroupID:     testGroupID,
		MaterialID:  testMaterialID,
	})

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Code != apperror.ErrCodeInternalError {
		t.Errorf("expected INTERNAL_ERROR from member provider failure, got %v", err)
	}
}

func TestUnpinMaterial_GroupProviderError_Returns500(t *testing.T) {
	uc := newUnpinMaterialUseCase(&mockMaterialRepository{}, groupProviderError(), notLead())

	_, err := uc.Execute(context.Background(), UnpinMaterialInput{
		CurrentUser: asCoach(testAuthorID),
		GroupID:     testGroupID,
		MaterialID:  testMaterialID,
	})

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Code != apperror.ErrCodeInternalError {
		t.Errorf("expected INTERNAL_ERROR from group provider failure, got %v", err)
	}
}

func TestUnpinMaterial_GroupNotFound(t *testing.T) {
	uc := newUnpinMaterialUseCase(&mockMaterialRepository{}, groupNotFound(), notLead())

	_, err := uc.Execute(context.Background(), UnpinMaterialInput{
		CurrentUser: asCoach(testAuthorID),
		GroupID:     testGroupID,
		MaterialID:  testMaterialID,
	})

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Code != ErrCodeGroupNotFound {
		t.Errorf("expected GROUP_NOT_FOUND, got %v", err)
	}
}

func TestUnpinMaterial_MaterialInOtherGroup(t *testing.T) {
	m := newPinnedMaterial()
	uc := newUnpinMaterialUseCase(repoWith(m), groupExists(), notLead())

	_, err := uc.Execute(context.Background(), UnpinMaterialInput{
		CurrentUser: asCoach(testAuthorID),
		GroupID:     "other-group-id",
		MaterialID:  testMaterialID,
	})

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Code != domainMaterial.ErrCodeMaterialNotFound {
		t.Errorf("expected MATERIAL_NOT_FOUND for cross-group access, got %v", err)
	}
}

func TestUnpinMaterial_SaveError(t *testing.T) {
	m := newPinnedMaterial()
	repo := &mockMaterialRepository{
		findByIDFn: func(_ context.Context, _ string) (*domainMaterial.Material, error) { return m, nil },
		saveFn:     func(_ context.Context, _ *domainMaterial.Material) error { return apperror.NewInternal() },
	}
	uc := newUnpinMaterialUseCase(repo, groupExists(), notLead())

	_, err := uc.Execute(context.Background(), UnpinMaterialInput{
		CurrentUser: asCoach(testAuthorID),
		GroupID:     testGroupID,
		MaterialID:  testMaterialID,
	})

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Code != apperror.ErrCodeInternalError {
		t.Errorf("expected INTERNAL_ERROR from save failure, got %v", err)
	}
}
