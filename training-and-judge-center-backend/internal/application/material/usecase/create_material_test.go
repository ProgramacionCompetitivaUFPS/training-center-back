package usecase

import (
	"context"
	"errors"
	"testing"

	appMaterial "github.com/training-judge-center/backend/internal/application/material"
	"github.com/training-judge-center/backend/pkg/apperror"
)

func newCreateUC(repo *mockMaterialRepository, group *mockGroupProvider, member *mockGroupMemberProvider) *CreateMaterial {
	return NewCreateMaterial(repo, group, member)
}

func TestCreateMaterial_SuccessByLead(t *testing.T) {
	uc := newCreateUC(&mockMaterialRepository{}, groupExists(), isLead())

	out, err := uc.Execute(context.Background(), CreateMaterialInput{
		CurrentUser: asCoach(testAuthorID),
		GroupID:     testGroupID,
		Title:       "Hello World",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Material.Title().String() != "Hello World" {
		t.Errorf("expected title 'Hello World', got %q", out.Material.Title().String())
	}
	if out.Material.Status().String() != "DRAFT" {
		t.Errorf("expected DRAFT status, got %q", out.Material.Status().String())
	}
	if out.Material.AuthorID().Value() != testAuthorID {
		t.Errorf("expected authorID %q, got %q", testAuthorID, out.Material.AuthorID().Value())
	}
	if out.Material.GroupID() != testGroupID {
		t.Errorf("expected groupID %q, got %q", testGroupID, out.Material.GroupID())
	}
}

func TestCreateMaterial_SuccessByAdmin(t *testing.T) {
	uc := newCreateUC(&mockMaterialRepository{}, groupExists(), notLead())

	out, err := uc.Execute(context.Background(), CreateMaterialInput{
		CurrentUser: asAdmin(testAuthorID),
		GroupID:     testGroupID,
		Title:       "Admin Material",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Material == nil {
		t.Fatal("expected material, got nil")
	}
}

func TestCreateMaterial_GroupNotFound(t *testing.T) {
	uc := newCreateUC(&mockMaterialRepository{}, groupNotFound(), notLead())

	_, err := uc.Execute(context.Background(), CreateMaterialInput{
		CurrentUser: asCoach(testAuthorID),
		GroupID:     testGroupID,
		Title:       "Hello",
	})

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Code != appMaterial.ErrCodeGroupNotFound {
		t.Errorf("expected GROUP_NOT_FOUND, got %v", err)
	}
}

func TestCreateMaterial_ForbiddenIfNotLead(t *testing.T) {
	uc := newCreateUC(&mockMaterialRepository{}, groupExists(), notLead())

	_, err := uc.Execute(context.Background(), CreateMaterialInput{
		CurrentUser: asCoach(testAuthorID),
		GroupID:     testGroupID,
		Title:       "Hello",
	})

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Code != appMaterial.ErrCodeInsufficientPerms {
		t.Errorf("expected INSUFFICIENT_PERMISSIONS, got %v", err)
	}
}

func TestCreateMaterial_ForbiddenIfContestant(t *testing.T) {
	uc := newCreateUC(&mockMaterialRepository{}, groupExists(), notLead())

	_, err := uc.Execute(context.Background(), CreateMaterialInput{
		CurrentUser: asContestant(testAuthorID),
		GroupID:     testGroupID,
		Title:       "Hello",
	})

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Code != appMaterial.ErrCodeInsufficientPerms {
		t.Errorf("expected INSUFFICIENT_PERMISSIONS, got %v", err)
	}
}

func TestCreateMaterial_ValidationErrorEmptyTitle(t *testing.T) {
	uc := newCreateUC(&mockMaterialRepository{}, groupExists(), isLead())

	_, err := uc.Execute(context.Background(), CreateMaterialInput{
		CurrentUser: asCoach(testAuthorID),
		GroupID:     testGroupID,
		Title:       "",
	})

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Code != apperror.ErrCodeValidationError {
		t.Errorf("expected VALIDATION_ERROR, got %v", err)
	}
}

func TestCreateMaterial_ValidationErrorInvalidTags(t *testing.T) {
	uc := newCreateUC(&mockMaterialRepository{}, groupExists(), isLead())

	_, err := uc.Execute(context.Background(), CreateMaterialInput{
		CurrentUser: asCoach(testAuthorID),
		GroupID:     testGroupID,
		Title:       "Valid Title",
		Tags:        []string{"INVALID TAG!"},
	})

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Code != apperror.ErrCodeValidationError {
		t.Errorf("expected VALIDATION_ERROR, got %v", err)
	}
}

func TestCreateMaterial_DefaultsContentAndTags(t *testing.T) {
	uc := newCreateUC(&mockMaterialRepository{}, groupExists(), isLead())

	out, err := uc.Execute(context.Background(), CreateMaterialInput{
		CurrentUser: asCoach(testAuthorID),
		GroupID:     testGroupID,
		Title:       "Minimal",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Material.Content().String() != "" {
		t.Errorf("expected empty content, got %q", out.Material.Content().String())
	}
	if len(out.Material.Tags().Values()) != 0 {
		t.Errorf("expected empty tags, got %v", out.Material.Tags().Values())
	}
}
