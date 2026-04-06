package user

import (
	"context"
	"errors"
	"testing"

	domain "github.com/training-judge-center/backend/internal/domain/user"
	"github.com/training-judge-center/backend/pkg/apperror"
)

func validInput() CreateUserInput {
	return CreateUserInput{
		Email:       "test@example.com",
		Password:    "Secret1!",
		Name:        "Test User",
		Nickname:    "testuser",
		Country:     "Colombia",
		City:        "Cúcuta",
		Institution: "UFPS",
	}
}

func TestCreateUser_Success(t *testing.T) {
	repo := newNoConflictRepo()
	uc := NewCreateUserUseCase(repo)

	result, err := uc.Execute(context.Background(), validInput())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result == nil {
		t.Fatal("expected user, got nil")
	}
	if result.Email().String() != "test@example.com" {
		t.Errorf("expected email %q, got %q", "test@example.com", result.Email().String())
	}
	if result.Nickname().String() != "testuser" {
		t.Errorf("expected nickname %q, got %q", "testuser", result.Nickname().String())
	}
	if result.Role().String() != "CONTESTANT" {
		t.Errorf("expected role CONTESTANT, got %q", result.Role().String())
	}
	if result.Status().String() != "ACTIVE" {
		t.Errorf("expected status ACTIVE, got %q", result.Status().String())
	}
	if result.ID() == "" {
		t.Error("expected non-empty ID")
	}
}

func TestCreateUser_ValidationErrors(t *testing.T) {
	repo := newNoConflictRepo()
	uc := NewCreateUserUseCase(repo)

	input := CreateUserInput{
		Email:       "",
		Password:    "weak",
		Name:        "",
		Nickname:    "",
		Country:     "",
		City:        "",
		Institution: "",
	}

	_, err := uc.Execute(context.Background(), input)
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
	if len(appErr.Details) < 7 {
		t.Errorf("expected at least 7 field errors (got %d): all invalid fields should be reported at once", len(appErr.Details))
	}
}

func TestCreateUser_EmailAlreadyExists(t *testing.T) {
	repo := newNoConflictRepo()
	repo.saveFn = func(_ context.Context, _ *domain.User) error {
		return domain.ErrEmailConflict
	}
	uc := NewCreateUserUseCase(repo)

	_, err := uc.Execute(context.Background(), validInput())
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	appErr, ok := err.(*apperror.AppError)
	if !ok {
		t.Fatalf("expected *apperror.AppError, got %T", err)
	}
	if appErr.Code != "EMAIL_ALREADY_EXISTS" {
		t.Errorf("expected code EMAIL_ALREADY_EXISTS, got %q", appErr.Code)
	}
}

func TestCreateUser_NicknameAlreadyExists(t *testing.T) {
	repo := newNoConflictRepo()
	repo.saveFn = func(_ context.Context, _ *domain.User) error {
		return domain.ErrNicknameConflict
	}
	uc := NewCreateUserUseCase(repo)

	_, err := uc.Execute(context.Background(), validInput())
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

func TestCreateUser_RepositorySaveError(t *testing.T) {
	repo := newNoConflictRepo()
	repo.saveFn = func(_ context.Context, _ *domain.User) error {
		return errors.New("db connection lost")
	}
	uc := NewCreateUserUseCase(repo)

	_, err := uc.Execute(context.Background(), validInput())
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

