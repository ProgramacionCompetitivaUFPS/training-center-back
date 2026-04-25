package usecase

import (
	"context"
	"errors"
	"testing"

	appMaterial "github.com/training-judge-center/backend/internal/application/material"
	domainMaterial "github.com/training-judge-center/backend/internal/domain/material"
	"github.com/training-judge-center/backend/pkg/apperror"
)

func newUpdateUC(repo *mockMaterialRepository, group *mockGroupProvider) *UpdateMaterial {
	return NewUpdateMaterial(repo, group)
}

func TestUpdateMaterial_SuccessByAuthor(t *testing.T) {
	m := newTestMaterial()
	uc := newUpdateUC(repoWith(m), groupExists())

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
	if out.Material.Title().String() != "Updated Title" {
		t.Errorf("expected updated title, got %q", out.Material.Title().String())
	}
}

func TestUpdateMaterial_SuccessByAdmin(t *testing.T) {
	m := newTestMaterial()
	uc := newUpdateUC(repoWith(m), groupExists())

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
	if out.Material.Title().String() != "Admin Update" {
		t.Errorf("expected 'Admin Update', got %q", out.Material.Title().String())
	}
}

func TestUpdateMaterial_ForbiddenIfNotAuthor(t *testing.T) {
	m := newTestMaterial()
	uc := newUpdateUC(repoWith(m), groupExists())

	newTitle := "Sneaky Update"
	_, err := uc.Execute(context.Background(), UpdateMaterialInput{
		CurrentUser: asCoach(testOtherID),
		GroupID:     testGroupID,
		MaterialID:  testMaterialID,
		Title:       &newTitle,
	})

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Code != appMaterial.ErrCodeNotMaterialAuthor {
		t.Errorf("expected NOT_MATERIAL_AUTHOR, got %v", err)
	}
}

func TestUpdateMaterial_GroupNotFound(t *testing.T) {
	uc := newUpdateUC(&mockMaterialRepository{}, groupNotFound())

	newTitle := "Title"
	_, err := uc.Execute(context.Background(), UpdateMaterialInput{
		CurrentUser: asCoach(testAuthorID),
		GroupID:     testGroupID,
		MaterialID:  testMaterialID,
		Title:       &newTitle,
	})

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Code != appMaterial.ErrCodeGroupNotFound {
		t.Errorf("expected GROUP_NOT_FOUND, got %v", err)
	}
}

func TestUpdateMaterial_MaterialNotFound(t *testing.T) {
	uc := newUpdateUC(&mockMaterialRepository{}, groupExists())

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
	uc := newUpdateUC(repoWith(m), groupExists())

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
	uc := newUpdateUC(repoWith(m), groupExists())

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
	if out.Material.Title().String() != "Only Title" {
		t.Errorf("expected 'Only Title', got %q", out.Material.Title().String())
	}
	if out.Material.Content().String() != "" {
		t.Error("content should remain unchanged")
	}
}

func TestUpdateMaterial_ClearTags(t *testing.T) {
	m := newTestMaterial()
	uc := newUpdateUC(repoWith(m), groupExists())

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
	if len(out.Material.Tags().Values()) != 0 {
		t.Errorf("expected empty tags, got %v", out.Material.Tags().Values())
	}
}

func TestUpdateMaterial_ClearContent(t *testing.T) {
	m := newTestMaterial()
	uc := newUpdateUC(repoWith(m), groupExists())

	emptyContent := ""
	out, err := uc.Execute(context.Background(), UpdateMaterialInput{
		CurrentUser: asCoach(testAuthorID),
		GroupID:     testGroupID,
		MaterialID:  testMaterialID,
		Content:     &emptyContent,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Material.Content().String() != "" {
		t.Errorf("expected empty content, got %q", out.Material.Content().String())
	}
}

func TestUpdateMaterial_ImmutableFields(t *testing.T) {
	m := newTestMaterial()
	uc := newUpdateUC(repoWith(m), groupExists())

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
	if out.Material.Status().String() != "DRAFT" {
		t.Error("status should remain unchanged")
	}
	if out.Material.Pinned() {
		t.Error("pinned should remain unchanged")
	}
	if out.Material.AuthorID().Value() != testAuthorID {
		t.Error("authorID should remain unchanged")
	}
	if out.Material.GroupID() != testGroupID {
		t.Error("groupID should remain unchanged")
	}
}

func TestUpdateMaterial_ValidationErrorEmptyTitle(t *testing.T) {
	m := newTestMaterial()
	uc := newUpdateUC(repoWith(m), groupExists())

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
