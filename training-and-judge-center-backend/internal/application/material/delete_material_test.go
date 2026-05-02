package material

import (
	"context"
	"errors"
	"testing"

	domainMaterial "github.com/training-judge-center/backend/internal/domain/material"
	"github.com/training-judge-center/backend/pkg/apperror"
)

func newDeleteUC(repo *mockMaterialRepository, group *mockGroupProvider) *DeleteMaterial {
	return NewDeleteMaterial(repo, group)
}

func TestDeleteMaterial_SuccessByAuthor(t *testing.T) {
	m := newTestMaterial()
	deleted := false
	repo := &mockMaterialRepository{
		findByIDFn: func(_ context.Context, _ string) (*domainMaterial.Material, error) { return m, nil },
		deleteFn:   func(_ context.Context, _ string) error { deleted = true; return nil },
	}
	uc := newDeleteUC(repo, groupExists())

	err := uc.Execute(context.Background(), DeleteMaterialInput{
		CurrentUser: asCoach(testAuthorID),
		GroupID:     testGroupID,
		MaterialID:  testMaterialID,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !deleted {
		t.Error("expected repo.Delete to be called")
	}
}

func TestDeleteMaterial_SuccessByAdmin(t *testing.T) {
	m := newTestMaterial()
	deleted := false
	repo := &mockMaterialRepository{
		findByIDFn: func(_ context.Context, _ string) (*domainMaterial.Material, error) { return m, nil },
		deleteFn:   func(_ context.Context, _ string) error { deleted = true; return nil },
	}
	uc := newDeleteUC(repo, groupExists())

	err := uc.Execute(context.Background(), DeleteMaterialInput{
		CurrentUser: asAdmin(testOtherID),
		GroupID:     testGroupID,
		MaterialID:  testMaterialID,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !deleted {
		t.Error("expected repo.Delete to be called for admin")
	}
}

func TestDeleteMaterial_SuccessPublishedMaterial(t *testing.T) {
	m := newPublishedMaterial()
	uc := newDeleteUC(repoWith(m), groupExists())

	err := uc.Execute(context.Background(), DeleteMaterialInput{
		CurrentUser: asCoach(testAuthorID),
		GroupID:     testGroupID,
		MaterialID:  testMaterialID,
	})

	if err != nil {
		t.Fatalf("expected published material to be deletable, got: %v", err)
	}
}

func TestDeleteMaterial_SuccessPinnedMaterial(t *testing.T) {
	m := newPinnedMaterial()
	uc := newDeleteUC(repoWith(m), groupExists())

	err := uc.Execute(context.Background(), DeleteMaterialInput{
		CurrentUser: asCoach(testAuthorID),
		GroupID:     testGroupID,
		MaterialID:  testMaterialID,
	})

	if err != nil {
		t.Fatalf("expected pinned material to be deletable, got: %v", err)
	}
}

func TestDeleteMaterial_Forbidden_NonAuthorLead(t *testing.T) {
	m := newTestMaterial()
	uc := newDeleteUC(repoWith(m), groupExists())

	err := uc.Execute(context.Background(), DeleteMaterialInput{
		CurrentUser: asCoach(testOtherID),
		GroupID:     testGroupID,
		MaterialID:  testMaterialID,
	})

	var notAuthorErr *NotMaterialAuthorError
	if !errors.As(err, &notAuthorErr) {
		t.Fatalf("expected NotMaterialAuthorError, got %v", err)
	}
	if notAuthorErr.AuthorID != testAuthorID {
		t.Errorf("expected authorId %q, got %q", testAuthorID, notAuthorErr.AuthorID)
	}
	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Code != ErrCodeNotMaterialAuthor {
		t.Errorf("expected NOT_MATERIAL_AUTHOR app error code, got %v", err)
	}
}

func TestDeleteMaterial_GroupNotFound(t *testing.T) {
	uc := newDeleteUC(&mockMaterialRepository{}, groupNotFound())

	err := uc.Execute(context.Background(), DeleteMaterialInput{
		CurrentUser: asCoach(testAuthorID),
		GroupID:     testGroupID,
		MaterialID:  testMaterialID,
	})

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Code != ErrCodeGroupNotFound {
		t.Errorf("expected GROUP_NOT_FOUND, got %v", err)
	}
}

func TestDeleteMaterial_MaterialNotFound(t *testing.T) {
	uc := newDeleteUC(&mockMaterialRepository{}, groupExists())

	err := uc.Execute(context.Background(), DeleteMaterialInput{
		CurrentUser: asCoach(testAuthorID),
		GroupID:     testGroupID,
		MaterialID:  "nonexistent",
	})

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Code != domainMaterial.ErrCodeMaterialNotFound {
		t.Errorf("expected MATERIAL_NOT_FOUND, got %v", err)
	}
}

func TestDeleteMaterial_MaterialInOtherGroup(t *testing.T) {
	m := newTestMaterial()
	uc := newDeleteUC(repoWith(m), groupExists())

	err := uc.Execute(context.Background(), DeleteMaterialInput{
		CurrentUser: asCoach(testAuthorID),
		GroupID:     "other-group-id",
		MaterialID:  testMaterialID,
	})

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Code != domainMaterial.ErrCodeMaterialNotFound {
		t.Errorf("expected MATERIAL_NOT_FOUND for cross-group access, got %v", err)
	}
}

func TestDeleteMaterial_GroupProviderError_Returns500(t *testing.T) {
	uc := newDeleteUC(&mockMaterialRepository{}, groupProviderError())

	err := uc.Execute(context.Background(), DeleteMaterialInput{
		CurrentUser: asCoach(testAuthorID),
		GroupID:     testGroupID,
		MaterialID:  testMaterialID,
	})

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Code != apperror.ErrCodeInternalError {
		t.Errorf("expected INTERNAL_ERROR from group provider failure, got %v", err)
	}
}

func TestDeleteMaterial_DeleteError_Returns500(t *testing.T) {
	m := newTestMaterial()
	repo := &mockMaterialRepository{
		findByIDFn: func(_ context.Context, _ string) (*domainMaterial.Material, error) { return m, nil },
		deleteFn:   func(_ context.Context, _ string) error { return errors.New("db error") },
	}
	uc := newDeleteUC(repo, groupExists())

	err := uc.Execute(context.Background(), DeleteMaterialInput{
		CurrentUser: asCoach(testAuthorID),
		GroupID:     testGroupID,
		MaterialID:  testMaterialID,
	})

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Code != apperror.ErrCodeInternalError {
		t.Errorf("expected INTERNAL_ERROR from delete failure, got %v", err)
	}
}
