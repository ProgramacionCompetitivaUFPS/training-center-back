package material

import (
	"context"
	"errors"
	"testing"

	domainMaterial "github.com/training-judge-center/backend/internal/domain/material"
	"github.com/training-judge-center/backend/pkg/apperror"
)

func newUnpublishUC(repo *mockMaterialRepository, group *mockGroupProvider) *UnpublishMaterialUseCase {
	return NewUnpublishMaterialUseCase(repo, group, stubAuthorProvider())
}

func TestUnpublishMaterial_SuccessByAuthor(t *testing.T) {
	m := newPublishedMaterial()
	saved := false
	repo := &mockMaterialRepository{
		findByIDFn: func(_ context.Context, _ string) (*domainMaterial.Material, error) { return m, nil },
		saveFn:     func(_ context.Context, _ *domainMaterial.Material) error { saved = true; return nil },
	}
	uc := newUnpublishUC(repo, groupExists())

	out, err := uc.Execute(context.Background(), UnpublishMaterialInput{
		CurrentUser: asCoach(testAuthorID),
		GroupID:     testGroupID,
		MaterialID:  testMaterialID,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Material.Status != "DRAFT" {
		t.Errorf("expected DRAFT, got %q", out.Material.Status)
	}
	if !saved {
		t.Error("expected repo.Save to be called")
	}
}

func TestUnpublishMaterial_SuccessByAdmin(t *testing.T) {
	m := newPublishedMaterial()
	uc := newUnpublishUC(repoWith(m), groupExists())

	out, err := uc.Execute(context.Background(), UnpublishMaterialInput{
		CurrentUser: asAdmin(testOtherID),
		GroupID:     testGroupID,
		MaterialID:  testMaterialID,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Material.Status != "DRAFT" {
		t.Errorf("expected DRAFT, got %q", out.Material.Status)
	}
}

func TestUnpublishMaterial_Idempotent_AlreadyDraft(t *testing.T) {
	m := newTestMaterial()
	saveCalled := false
	repo := &mockMaterialRepository{
		findByIDFn: func(_ context.Context, _ string) (*domainMaterial.Material, error) { return m, nil },
		saveFn:     func(_ context.Context, _ *domainMaterial.Material) error { saveCalled = true; return nil },
	}
	uc := newUnpublishUC(repo, groupExists())

	out, err := uc.Execute(context.Background(), UnpublishMaterialInput{
		CurrentUser: asCoach(testAuthorID),
		GroupID:     testGroupID,
		MaterialID:  testMaterialID,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Material.Status != "DRAFT" {
		t.Errorf("expected DRAFT, got %q", out.Material.Status)
	}
	if saveCalled {
		t.Error("repo.Save must NOT be called for already-draft material (idempotent)")
	}
}

func TestUnpublishMaterial_AutoUnpin(t *testing.T) {
	m := newPinnedMaterial()
	uc := newUnpublishUC(repoWith(m), groupExists())

	out, err := uc.Execute(context.Background(), UnpublishMaterialInput{
		CurrentUser: asCoach(testAuthorID),
		GroupID:     testGroupID,
		MaterialID:  testMaterialID,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Material.Pinned {
		t.Error("expected pinned=false after unpublish")
	}
	if out.Material.PinnedAt != nil {
		t.Error("expected pinnedAt=nil after unpublish")
	}
}

func TestUnpublishMaterial_PreservesPublishedAt(t *testing.T) {
	m := newPublishedMaterial()
	originalPublishedAt := m.PublishedAt()
	uc := newUnpublishUC(repoWith(m), groupExists())

	out, err := uc.Execute(context.Background(), UnpublishMaterialInput{
		CurrentUser: asCoach(testAuthorID),
		GroupID:     testGroupID,
		MaterialID:  testMaterialID,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Material.PublishedAt == nil || !out.Material.PublishedAt.Equal(*originalPublishedAt) {
		t.Error("publishedAt must be preserved after unpublish")
	}
}

func TestUnpublishMaterial_Forbidden_NonAuthorLead(t *testing.T) {
	m := newPublishedMaterial()
	uc := newUnpublishUC(repoWith(m), groupExists())

	_, err := uc.Execute(context.Background(), UnpublishMaterialInput{
		CurrentUser: asCoach(testOtherID),
		GroupID:     testGroupID,
		MaterialID:  testMaterialID,
	})

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Code != ErrCodeNotMaterialAuthor {
		t.Errorf("expected NOT_MATERIAL_AUTHOR, got %v", err)
	}
}

func TestUnpublishMaterial_GroupProviderError_Returns500(t *testing.T) {
	uc := newUnpublishUC(&mockMaterialRepository{}, groupProviderError())

	_, err := uc.Execute(context.Background(), UnpublishMaterialInput{
		CurrentUser: asCoach(testAuthorID),
		GroupID:     testGroupID,
		MaterialID:  testMaterialID,
	})

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Code != apperror.ErrCodeInternalError {
		t.Errorf("expected INTERNAL_ERROR from group provider failure, got %v", err)
	}
}

func TestUnpublishMaterial_SaveError(t *testing.T) {
	m := newPublishedMaterial()
	repo := &mockMaterialRepository{
		findByIDFn: func(_ context.Context, _ string) (*domainMaterial.Material, error) { return m, nil },
		saveFn:     func(_ context.Context, _ *domainMaterial.Material) error { return errors.New("db error") },
	}
	uc := newUnpublishUC(repo, groupExists())

	_, err := uc.Execute(context.Background(), UnpublishMaterialInput{
		CurrentUser: asCoach(testAuthorID),
		GroupID:     testGroupID,
		MaterialID:  testMaterialID,
	})

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Code != apperror.ErrCodeInternalError {
		t.Errorf("expected INTERNAL_ERROR from save failure, got %v", err)
	}
}

func TestUnpublishMaterial_GroupNotFound(t *testing.T) {
	uc := newUnpublishUC(&mockMaterialRepository{}, groupNotFound())

	_, err := uc.Execute(context.Background(), UnpublishMaterialInput{
		CurrentUser: asCoach(testAuthorID),
		GroupID:     testGroupID,
		MaterialID:  testMaterialID,
	})

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Code != ErrCodeGroupNotFound {
		t.Errorf("expected GROUP_NOT_FOUND, got %v", err)
	}
}

func TestUnpublishMaterial_MaterialNotFound(t *testing.T) {
	uc := newUnpublishUC(&mockMaterialRepository{}, groupExists())

	_, err := uc.Execute(context.Background(), UnpublishMaterialInput{
		CurrentUser: asCoach(testAuthorID),
		GroupID:     testGroupID,
		MaterialID:  "nonexistent",
	})

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Code != domainMaterial.ErrCodeMaterialNotFound {
		t.Errorf("expected MATERIAL_NOT_FOUND, got %v", err)
	}
}

func TestUnpublishMaterial_MaterialInOtherGroup(t *testing.T) {
	m := newPublishedMaterial()
	uc := newUnpublishUC(repoWith(m), groupExists())

	_, err := uc.Execute(context.Background(), UnpublishMaterialInput{
		CurrentUser: asCoach(testAuthorID),
		GroupID:     "other-group-id",
		MaterialID:  testMaterialID,
	})

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Code != domainMaterial.ErrCodeMaterialNotFound {
		t.Errorf("expected MATERIAL_NOT_FOUND for cross-group access, got %v", err)
	}
}
