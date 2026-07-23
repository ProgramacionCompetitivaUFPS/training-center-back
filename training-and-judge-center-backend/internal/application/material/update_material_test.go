package material

import (
	"context"
	"errors"
	"testing"

	domainMaterial "github.com/training-judge-center/backend/internal/domain/material"
	"github.com/training-judge-center/backend/pkg/apperror"
)

func newUpdateMaterialUseCase(repo *mockMaterialRepository, group *mockGroupProvider) *UpdateMaterialUseCase {
	return NewUpdateMaterialUseCase(repo, group, stubAuthorProvider())
}

func TestUpdateMaterial_SuccessByAuthor(t *testing.T) {
	m := newTestMaterial()
	uc := newUpdateMaterialUseCase(repoWith(m), groupExists())

	newTitle := "Updated Title"
	out, err := uc.Execute(context.Background(), UpdateMaterialInput{
		CurrentUser: asCoach(testAuthorID),
		GroupID:     testGroupID,
		MaterialID:  testMaterialID,
		Title:       &newTitle,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Material.Title != "Updated Title" {
		t.Errorf("expected updated title, got %q", out.Material.Title)
	}
}

func TestUpdateMaterial_SuccessByAdmin(t *testing.T) {
	m := newTestMaterial()
	uc := newUpdateMaterialUseCase(repoWith(m), groupExists())

	newTitle := "Admin Update"
	out, err := uc.Execute(context.Background(), UpdateMaterialInput{
		CurrentUser: asAdmin(testOtherID),
		GroupID:     testGroupID,
		MaterialID:  testMaterialID,
		Title:       &newTitle,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Material.Title != "Admin Update" {
		t.Errorf("expected 'Admin Update', got %q", out.Material.Title)
	}
}

func TestUpdateMaterial_ForbiddenIfNotAuthor(t *testing.T) {
	m := newTestMaterial()
	uc := newUpdateMaterialUseCase(repoWith(m), groupExists())

	newTitle := "Sneaky Update"
	_, err := uc.Execute(context.Background(), UpdateMaterialInput{
		CurrentUser: asCoach(testOtherID),
		GroupID:     testGroupID,
		MaterialID:  testMaterialID,
		Title:       &newTitle,
	})

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Code != ErrCodeNotMaterialAuthor {
		t.Errorf("expected NOT_MATERIAL_AUTHOR, got %v", err)
	}
}

func TestUpdateMaterial_GroupProviderError(t *testing.T) {
	group := &mockGroupProvider{
		existsFn: func(_ context.Context, _ string) (bool, error) {
			return false, apperror.NewInternal()
		},
	}
	uc := newUpdateMaterialUseCase(&mockMaterialRepository{}, group)
	newTitle := "Title"
	_, err := uc.Execute(context.Background(), UpdateMaterialInput{
		CurrentUser: asCoach(testAuthorID),
		GroupID:     testGroupID,
		MaterialID:  testMaterialID,
		Title:       &newTitle,
	})
	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Code != apperror.ErrCodeInternalError {
		t.Errorf("expected INTERNAL_ERROR from group provider error, got %v", err)
	}
}

func TestUpdateMaterial_GroupNotFound(t *testing.T) {
	uc := newUpdateMaterialUseCase(&mockMaterialRepository{}, groupNotFound())

	newTitle := "Title"
	_, err := uc.Execute(context.Background(), UpdateMaterialInput{
		CurrentUser: asCoach(testAuthorID),
		GroupID:     testGroupID,
		MaterialID:  testMaterialID,
		Title:       &newTitle,
	})

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Code != ErrCodeGroupNotFound {
		t.Errorf("expected GROUP_NOT_FOUND, got %v", err)
	}
}

func TestUpdateMaterial_FindByIDInternalError(t *testing.T) {
	repo := &mockMaterialRepository{
		findByIDFn: func(_ context.Context, _ string) (*domainMaterial.Material, error) {
			return nil, apperror.NewInternal()
		},
	}
	uc := newUpdateMaterialUseCase(repo, groupExists())
	newTitle := "Title"
	_, err := uc.Execute(context.Background(), UpdateMaterialInput{
		CurrentUser: asCoach(testAuthorID),
		GroupID:     testGroupID,
		MaterialID:  testMaterialID,
		Title:       &newTitle,
	})
	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Code != apperror.ErrCodeInternalError {
		t.Errorf("expected INTERNAL_ERROR from FindByID failure, got %v", err)
	}
}

func TestUpdateMaterial_MaterialNotFound(t *testing.T) {
	uc := newUpdateMaterialUseCase(&mockMaterialRepository{}, groupExists())

	newTitle := "Title"
	_, err := uc.Execute(context.Background(), UpdateMaterialInput{
		CurrentUser: asCoach(testAuthorID),
		GroupID:     testGroupID,
		MaterialID:  "nonexistent",
		Title:       &newTitle,
	})

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Code != domainMaterial.ErrCodeMaterialNotFound {
		t.Errorf("expected MATERIAL_NOT_FOUND, got %v", err)
	}
}

func TestUpdateMaterial_MaterialNotInGroup(t *testing.T) {
	m := newTestMaterial()
	uc := newUpdateMaterialUseCase(repoWith(m), groupExists())

	newTitle := "Title"
	_, err := uc.Execute(context.Background(), UpdateMaterialInput{
		CurrentUser: asCoach(testAuthorID),
		GroupID:     "different-group-id",
		MaterialID:  testMaterialID,
		Title:       &newTitle,
	})

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Code != domainMaterial.ErrCodeMaterialNotFound {
		t.Errorf("expected MATERIAL_NOT_FOUND, got %v", err)
	}
}

func TestUpdateMaterial_PartialUpdate(t *testing.T) {
	m := newTestMaterial()
	uc := newUpdateMaterialUseCase(repoWith(m), groupExists())

	newTitle := "Only Title"
	out, err := uc.Execute(context.Background(), UpdateMaterialInput{
		CurrentUser: asCoach(testAuthorID),
		GroupID:     testGroupID,
		MaterialID:  testMaterialID,
		Title:       &newTitle,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Material.Title != "Only Title" {
		t.Errorf("expected 'Only Title', got %q", out.Material.Title)
	}
	if out.Material.Content != "" {
		t.Error("content should remain unchanged")
	}
}

func TestUpdateMaterial_ClearTags(t *testing.T) {
	m := newTestMaterial()
	uc := newUpdateMaterialUseCase(repoWith(m), groupExists())

	emptyTags := []string{}
	out, err := uc.Execute(context.Background(), UpdateMaterialInput{
		CurrentUser: asCoach(testAuthorID),
		GroupID:     testGroupID,
		MaterialID:  testMaterialID,
		Tags:        &emptyTags,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out.Material.Tags) != 0 {
		t.Errorf("expected empty tags, got %v", out.Material.Tags)
	}
}

func TestUpdateMaterial_EmptyContentReturnsValidation(t *testing.T) {
	m := newTestMaterial()
	uc := newUpdateMaterialUseCase(repoWith(m), groupExists())

	emptyContent := ""
	_, err := uc.Execute(context.Background(), UpdateMaterialInput{
		CurrentUser: asCoach(testAuthorID),
		GroupID:     testGroupID,
		MaterialID:  testMaterialID,
		Content:     &emptyContent,
	})

	ae, ok := err.(*apperror.AppError)
	if !ok || ae.Code != apperror.ErrCodeValidationError {
		t.Fatalf("expected VALIDATION_ERROR, got %v", err)
	}
}

func TestUpdateMaterial_ImmutableFields(t *testing.T) {
	m := newTestMaterial()
	uc := newUpdateMaterialUseCase(repoWith(m), groupExists())

	newTitle := "New Title"
	out, err := uc.Execute(context.Background(), UpdateMaterialInput{
		CurrentUser: asCoach(testAuthorID),
		GroupID:     testGroupID,
		MaterialID:  testMaterialID,
		Title:       &newTitle,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Material.Status != "DRAFT" {
		t.Error("status should remain unchanged")
	}
	if out.Material.Pinned {
		t.Error("pinned should remain unchanged")
	}
	if out.Material.AuthorID != testAuthorID {
		t.Error("authorID should remain unchanged")
	}
	if out.Material.GroupID != testGroupID {
		t.Error("groupID should remain unchanged")
	}
}

func TestUpdateMaterial_SaveRepositoryError(t *testing.T) {
	m := newTestMaterial()
	repo := &mockMaterialRepository{
		findByIDFn: func(_ context.Context, _ string) (*domainMaterial.Material, error) {
			return m, nil
		},
		saveFn: func(_ context.Context, _ *domainMaterial.Material) error {
			return apperror.NewInternal()
		},
	}
	uc := newUpdateMaterialUseCase(repo, groupExists())
	newTitle := "Updated"
	_, err := uc.Execute(context.Background(), UpdateMaterialInput{
		CurrentUser: asCoach(testAuthorID),
		GroupID:     testGroupID,
		MaterialID:  testMaterialID,
		Title:       &newTitle,
	})
	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Code != apperror.ErrCodeInternalError {
		t.Errorf("expected INTERNAL_ERROR from save failure, got %v", err)
	}
}

func TestUpdateMaterial_NoOpAllNil_StillSavesAndUpdatesTimestamp(t *testing.T) {
	m := newTestMaterial()
	saved := false
	repo := &mockMaterialRepository{
		findByIDFn: func(_ context.Context, _ string) (*domainMaterial.Material, error) {
			return m, nil
		},
		saveFn: func(_ context.Context, _ *domainMaterial.Material) error {
			saved = true
			return nil
		},
	}
	uc := newUpdateMaterialUseCase(repo, groupExists())

	_, err := uc.Execute(context.Background(), UpdateMaterialInput{
		CurrentUser: asCoach(testAuthorID),
		GroupID:     testGroupID,
		MaterialID:  testMaterialID,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !saved {
		t.Error("expected repo.Save to be called even for a no-op update")
	}
}

func TestUpdateMaterial_ValidationErrorEmptyTitle(t *testing.T) {
	m := newTestMaterial()
	uc := newUpdateMaterialUseCase(repoWith(m), groupExists())

	emptyTitle := ""
	_, err := uc.Execute(context.Background(), UpdateMaterialInput{
		CurrentUser: asCoach(testAuthorID),
		GroupID:     testGroupID,
		MaterialID:  testMaterialID,
		Title:       &emptyTitle,
	})

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Code != apperror.ErrCodeValidationError {
		t.Errorf("expected VALIDATION_ERROR, got %v", err)
	}
}
