package material

import (
	"context"
	"errors"
	"testing"

	domainMaterial "github.com/training-judge-center/backend/internal/domain/material"
	"github.com/training-judge-center/backend/pkg/apperror"
)

func newPublishUC(repo *mockMaterialRepository, group *mockGroupProvider) *PublishMaterialUseCase {
	return NewPublishMaterialUseCase(repo, group, stubAuthorProvider())
}

func TestPublishMaterial_SuccessByAuthor(t *testing.T) {
	m := newTestMaterial()
	saved := false
	repo := &mockMaterialRepository{
		findByIDFn: func(_ context.Context, _ string) (*domainMaterial.Material, error) { return m, nil },
		saveFn:     func(_ context.Context, _ *domainMaterial.Material) error { saved = true; return nil },
	}
	uc := newPublishUC(repo, groupExists())

	out, err := uc.Execute(context.Background(), PublishMaterialInput{
		CurrentUser: asCoach(testAuthorID),
		GroupID:     testGroupID,
		MaterialID:  testMaterialID,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Material.Status != "PUBLISHED" {
		t.Errorf("expected PUBLISHED, got %q", out.Material.Status)
	}
	if out.Material.PublishedAt == nil {
		t.Error("expected publishedAt to be set")
	}
	if !saved {
		t.Error("expected repo.Save to be called")
	}
}

func TestPublishMaterial_SuccessByAdmin(t *testing.T) {
	m := newTestMaterial()
	uc := newPublishUC(repoWith(m), groupExists())

	out, err := uc.Execute(context.Background(), PublishMaterialInput{
		CurrentUser: asAdmin(testOtherID),
		GroupID:     testGroupID,
		MaterialID:  testMaterialID,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Material.Status != "PUBLISHED" {
		t.Errorf("expected PUBLISHED, got %q", out.Material.Status)
	}
}

func TestPublishMaterial_Idempotent_AlreadyPublished(t *testing.T) {
	m := newPublishedMaterial()
	originalPublishedAt := m.PublishedAt()
	saveCalled := false
	repo := &mockMaterialRepository{
		findByIDFn: func(_ context.Context, _ string) (*domainMaterial.Material, error) { return m, nil },
		saveFn:     func(_ context.Context, _ *domainMaterial.Material) error { saveCalled = true; return nil },
	}
	uc := newPublishUC(repo, groupExists())

	out, err := uc.Execute(context.Background(), PublishMaterialInput{
		CurrentUser: asCoach(testAuthorID),
		GroupID:     testGroupID,
		MaterialID:  testMaterialID,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Material.Status != "PUBLISHED" {
		t.Errorf("expected PUBLISHED, got %q", out.Material.Status)
	}
	if saveCalled {
		t.Error("repo.Save must NOT be called for already-published material (idempotent)")
	}
	if out.Material.PublishedAt == nil || !out.Material.PublishedAt.Equal(*originalPublishedAt) {
		t.Error("publishedAt must be preserved on idempotent publish")
	}
}

func TestPublishMaterial_Forbidden_NonAuthorLead(t *testing.T) {
	m := newTestMaterial()
	uc := newPublishUC(repoWith(m), groupExists())

	_, err := uc.Execute(context.Background(), PublishMaterialInput{
		CurrentUser: asCoach(testOtherID),
		GroupID:     testGroupID,
		MaterialID:  testMaterialID,
	})

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Code != ErrCodeNotMaterialAuthor {
		t.Errorf("expected NOT_MATERIAL_AUTHOR, got %v", err)
	}
}

func TestPublishMaterial_GroupNotFound(t *testing.T) {
	uc := newPublishUC(&mockMaterialRepository{}, groupNotFound())

	_, err := uc.Execute(context.Background(), PublishMaterialInput{
		CurrentUser: asCoach(testAuthorID),
		GroupID:     testGroupID,
		MaterialID:  testMaterialID,
	})

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Code != ErrCodeGroupNotFound {
		t.Errorf("expected GROUP_NOT_FOUND, got %v", err)
	}
}

func TestPublishMaterial_MaterialNotFound(t *testing.T) {
	uc := newPublishUC(&mockMaterialRepository{}, groupExists())

	_, err := uc.Execute(context.Background(), PublishMaterialInput{
		CurrentUser: asCoach(testAuthorID),
		GroupID:     testGroupID,
		MaterialID:  "nonexistent",
	})

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Code != domainMaterial.ErrCodeMaterialNotFound {
		t.Errorf("expected MATERIAL_NOT_FOUND, got %v", err)
	}
}

func TestPublishMaterial_MaterialInOtherGroup(t *testing.T) {
	m := newTestMaterial()
	uc := newPublishUC(repoWith(m), groupExists())

	_, err := uc.Execute(context.Background(), PublishMaterialInput{
		CurrentUser: asCoach(testAuthorID),
		GroupID:     "other-group-id",
		MaterialID:  testMaterialID,
	})

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Code != domainMaterial.ErrCodeMaterialNotFound {
		t.Errorf("expected MATERIAL_NOT_FOUND for cross-group access, got %v", err)
	}
}

func TestPublishMaterial_GroupProviderError_Returns500(t *testing.T) {
	uc := newPublishUC(&mockMaterialRepository{}, groupProviderError())

	_, err := uc.Execute(context.Background(), PublishMaterialInput{
		CurrentUser: asCoach(testAuthorID),
		GroupID:     testGroupID,
		MaterialID:  testMaterialID,
	})

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Code != apperror.ErrCodeInternalError {
		t.Errorf("expected INTERNAL_ERROR from group provider failure, got %v", err)
	}
}

func TestPublishMaterial_SaveError(t *testing.T) {
	m := newTestMaterial()
	repo := &mockMaterialRepository{
		findByIDFn: func(_ context.Context, _ string) (*domainMaterial.Material, error) { return m, nil },
		saveFn:     func(_ context.Context, _ *domainMaterial.Material) error { return errors.New("db error") },
	}
	uc := newPublishUC(repo, groupExists())

	_, err := uc.Execute(context.Background(), PublishMaterialInput{
		CurrentUser: asCoach(testAuthorID),
		GroupID:     testGroupID,
		MaterialID:  testMaterialID,
	})

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Code != apperror.ErrCodeInternalError {
		t.Errorf("expected INTERNAL_ERROR from save failure, got %v", err)
	}
}
