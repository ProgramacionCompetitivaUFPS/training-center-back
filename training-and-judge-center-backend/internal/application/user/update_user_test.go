package user

import (
	"context"
	"errors"
	"net/http"
	"testing"

	domain "github.com/training-judge-center/backend/internal/domain/user"
	"github.com/training-judge-center/backend/pkg/apperror"
)

func strPtr(s string) *string { return &s }

func TestUpdateUser_Success_AllFields(t *testing.T) {
	repo := newNoConflictRepo()
	activeUser := newUserWithRole("user-1", domain.RoleContestant, domain.StatusActive)
	repo.findByIDFn = func(_ context.Context, id string) (*domain.User, error) {
		if id == "user-1" {
			return activeUser, nil
		}
		return nil, nil
	}
	uc := NewUpdateUserUseCase(repo)

	result, err := uc.Execute(context.Background(), "user-1", UpdateUserInput{
		Name:        strPtr("Updated Name"),
		Nickname:    strPtr("updatednick"),
		Institution: strPtr("New University"),
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.Name != "Updated Name" {
		t.Errorf("expected name %q, got %q", "Updated Name", result.Name)
	}
	if result.Nickname.String() != "updatednick" {
		t.Errorf("expected nickname %q, got %q", "updatednick", result.Nickname.String())
	}
	if result.Institution != "New University" {
		t.Errorf("expected institution %q, got %q", "New University", result.Institution)
	}
	if result.UpdatedAt == nil {
		t.Error("expected updatedAt to be set")
	}
}

func TestUpdateUser_Success_PartialUpdate(t *testing.T) {
	repo := newNoConflictRepo()
	activeUser := newUserWithRole("user-1", domain.RoleContestant, domain.StatusActive)
	repo.findByIDFn = func(_ context.Context, id string) (*domain.User, error) {
		if id == "user-1" {
			return activeUser, nil
		}
		return nil, nil
	}
	uc := NewUpdateUserUseCase(repo)

	result, err := uc.Execute(context.Background(), "user-1", UpdateUserInput{
		Name: strPtr("Only Name Changed"),
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.Name != "Only Name Changed" {
		t.Errorf("expected name %q, got %q", "Only Name Changed", result.Name)
	}
	if result.Nickname.String() != "user-1" {
		t.Errorf("expected original nickname %q, got %q", "user-1", result.Nickname.String())
	}
}

func TestUpdateUser_NoFieldsProvided(t *testing.T) {
	repo := newNoConflictRepo()
	uc := NewUpdateUserUseCase(repo)

	_, err := uc.Execute(context.Background(), "user-1", UpdateUserInput{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	appErr, ok := err.(*apperror.AppError)
	if !ok {
		t.Fatalf("expected *apperror.AppError, got %T", err)
	}
	if appErr.Code != "VALIDATION_ERROR" {
		t.Errorf("expected code VALIDATION_ERROR, got %q", appErr.Code)
	}
	if appErr.StatusCode != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", appErr.StatusCode)
	}
}

func TestUpdateUser_UserNotFound(t *testing.T) {
	repo := newNoConflictRepo()
	uc := NewUpdateUserUseCase(repo)

	_, err := uc.Execute(context.Background(), "nonexistent", UpdateUserInput{
		Name: strPtr("New Name"),
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	appErr, ok := err.(*apperror.AppError)
	if !ok {
		t.Fatalf("expected *apperror.AppError, got %T", err)
	}
	if appErr.Code != "NOT_FOUND" {
		t.Errorf("expected code NOT_FOUND, got %q", appErr.Code)
	}
}

func TestUpdateUser_EmptyNameValidation(t *testing.T) {
	repo := newNoConflictRepo()
	activeUser := newUserWithRole("user-1", domain.RoleContestant, domain.StatusActive)
	repo.findByIDFn = func(_ context.Context, _ string) (*domain.User, error) {
		return activeUser, nil
	}
	uc := NewUpdateUserUseCase(repo)

	_, err := uc.Execute(context.Background(), "user-1", UpdateUserInput{
		Name: strPtr(""),
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	appErr, ok := err.(*apperror.AppError)
	if !ok {
		t.Fatalf("expected *apperror.AppError, got %T", err)
	}
	if appErr.Code != "VALIDATION_ERROR" {
		t.Errorf("expected code VALIDATION_ERROR, got %q", appErr.Code)
	}
}

func TestUpdateUser_NicknameAlreadyExists(t *testing.T) {
	repo := newNoConflictRepo()
	activeUser := newUserWithRole("user-1", domain.RoleContestant, domain.StatusActive)
	repo.findByIDFn = func(_ context.Context, _ string) (*domain.User, error) {
		return activeUser, nil
	}
	repo.existsByNicknameFn = func(_ context.Context, _ domain.Nickname) (bool, error) {
		return true, nil
	}
	uc := NewUpdateUserUseCase(repo)

	_, err := uc.Execute(context.Background(), "user-1", UpdateUserInput{
		Nickname: strPtr("taken-nick"),
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	appErr, ok := err.(*apperror.AppError)
	if !ok {
		t.Fatalf("expected *apperror.AppError, got %T", err)
	}
	if appErr.Code != "NICKNAME_ALREADY_EXISTS" {
		t.Errorf("expected code NICKNAME_ALREADY_EXISTS, got %q", appErr.Code)
	}
}

func TestUpdateUser_SameNicknameNoConflict(t *testing.T) {
	repo := newNoConflictRepo()
	activeUser := newUserWithRole("user-1", domain.RoleContestant, domain.StatusActive)
	repo.findByIDFn = func(_ context.Context, _ string) (*domain.User, error) {
		return activeUser, nil
	}
	uc := NewUpdateUserUseCase(repo)

	result, err := uc.Execute(context.Background(), "user-1", UpdateUserInput{
		Nickname: strPtr("user-1"),
	})
	if err != nil {
		t.Fatalf("expected no error when nickname unchanged, got %v", err)
	}
	if result.Nickname.String() != "user-1" {
		t.Errorf("expected nickname %q, got %q", "user-1", result.Nickname.String())
	}
}

func TestUpdateUser_RepositoryUpdateError(t *testing.T) {
	repo := newNoConflictRepo()
	activeUser := newUserWithRole("user-1", domain.RoleContestant, domain.StatusActive)
	repo.findByIDFn = func(_ context.Context, _ string) (*domain.User, error) {
		return activeUser, nil
	}
	repo.updateFn = func(_ context.Context, _ *domain.User) error {
		return errors.New("db connection lost")
	}
	uc := NewUpdateUserUseCase(repo)

	_, err := uc.Execute(context.Background(), "user-1", UpdateUserInput{
		Name: strPtr("New Name"),
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	appErr, ok := err.(*apperror.AppError)
	if !ok {
		t.Fatalf("expected *apperror.AppError, got %T", err)
	}
	if appErr.Code != "INTERNAL_ERROR" {
		t.Errorf("expected code INTERNAL_ERROR, got %q", appErr.Code)
	}
}

func TestUpdateUser_NicknameLowercased(t *testing.T) {
	repo := newNoConflictRepo()
	activeUser := newUserWithRole("user-1", domain.RoleContestant, domain.StatusActive)
	repo.findByIDFn = func(_ context.Context, _ string) (*domain.User, error) {
		return activeUser, nil
	}
	uc := NewUpdateUserUseCase(repo)

	result, err := uc.Execute(context.Background(), "user-1", UpdateUserInput{
		Nickname: strPtr("MyNewNick"),
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.Nickname.String() != "mynewnick" {
		t.Errorf("expected nickname to be lowercased %q, got %q", "mynewnick", result.Nickname.String())
	}
}
