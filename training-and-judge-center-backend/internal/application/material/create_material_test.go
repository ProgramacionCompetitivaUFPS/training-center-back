package material

import (
	"context"
	"errors"
	"testing"

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
	if out.Material.Title != "Hello World" {
		t.Errorf("expected title 'Hello World', got %q", out.Material.Title)
	}
	if out.Material.Status != "DRAFT" {
		t.Errorf("expected DRAFT status, got %q", out.Material.Status)
	}
	if out.Material.AuthorID != testAuthorID {
		t.Errorf("expected authorID %q, got %q", testAuthorID, out.Material.AuthorID)
	}
	if out.Material.GroupID != testGroupID {
		t.Errorf("expected groupID %q, got %q", testGroupID, out.Material.GroupID)
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
	if out == nil {
		t.Fatal("expected output, got nil")
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
	if !errors.As(err, &appErr) || appErr.Code != ErrCodeGroupNotFound {
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
	if !errors.As(err, &appErr) || appErr.Code != ErrCodeInsufficientPerms {
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
	if !errors.As(err, &appErr) || appErr.Code != ErrCodeInsufficientPerms {
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
	if out.Material.Content != "" {
		t.Errorf("expected empty content, got %q", out.Material.Content)
	}
	if len(out.Material.Tags) != 0 {
		t.Errorf("expected empty tags, got %v", out.Material.Tags)
	}
}
