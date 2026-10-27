package user

import (
	"context"
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
	uc := NewCreateUserUseCase(repo, newNoopGlobalGroupJoiner(), &mockTransactionManager{})

	result, err := uc.Execute(context.Background(), validInput())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.User.Email == nil || *result.User.Email != "test@example.com" {
		t.Errorf("expected email %q, got %v", "test@example.com", result.User.Email)
	}
	if result.User.Nickname != "testuser" {
		t.Errorf("expected nickname %q, got %q", "testuser", result.User.Nickname)
	}
	if result.User.Role != "CONTESTANT" {
		t.Errorf("expected role CONTESTANT, got %q", result.User.Role)
	}
	if result.User.Status != "ACTIVE" {
		t.Errorf("expected status ACTIVE, got %q", result.User.Status)
	}
	if result.User.ID == "" {
		t.Error("expected non-empty ID")
	}
}

func TestCreateUser_ValidationErrors(t *testing.T) {
	repo := newNoConflictRepo()
	uc := NewCreateUserUseCase(repo, newNoopGlobalGroupJoiner(), &mockTransactionManager{})

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
	if appErr.Code != apperror.ErrCodeValidationError {
		t.Errorf("expected code VALIDATION_ERROR, got %q", appErr.Code)
	}
	if len(appErr.Details) < 7 {
		t.Errorf("expected at least 7 field errors (got %d): all invalid fields should be reported at once", len(appErr.Details))
	}
}

func TestCreateUser_EmailAlreadyExists(t *testing.T) {
	repo := newNoConflictRepo()
	repo.saveFn = func(_ context.Context, _ *domain.User) error {
		return apperror.NewConflict(domain.ErrCodeEmailConflict, "email already in use")
	}
	uc := NewCreateUserUseCase(repo, newNoopGlobalGroupJoiner(), &mockTransactionManager{})

	_, err := uc.Execute(context.Background(), validInput())
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	appErr, ok := err.(*apperror.AppError)
	if !ok {
		t.Fatalf("expected *apperror.AppError, got %T", err)
	}
	if appErr.Code != domain.ErrCodeEmailConflict {
		t.Errorf("expected code %q, got %q", domain.ErrCodeEmailConflict, appErr.Code)
	}
}

func TestCreateUser_NicknameAlreadyExists(t *testing.T) {
	repo := newNoConflictRepo()
	repo.saveFn = func(_ context.Context, _ *domain.User) error {
		return apperror.NewConflict(domain.ErrCodeNicknameConflict, "nickname already in use")
	}
	uc := NewCreateUserUseCase(repo, newNoopGlobalGroupJoiner(), &mockTransactionManager{})

	_, err := uc.Execute(context.Background(), validInput())
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	appErr, ok := err.(*apperror.AppError)
	if !ok {
		t.Fatalf("expected *apperror.AppError, got %T", err)
	}
	if appErr.Code != domain.ErrCodeNicknameConflict {
		t.Errorf("expected code %q, got %q", domain.ErrCodeNicknameConflict, appErr.Code)
	}
}

func TestCreateUser_JoinsGlobalGroup(t *testing.T) {
	repo := newNoConflictRepo()
	joiner := newNoopGlobalGroupJoiner()
	var joinedUserID string
	joiner.addToGlobalGroupFn = func(_ context.Context, userID string) error {
		joinedUserID = userID
		return nil
	}
	uc := NewCreateUserUseCase(repo, joiner, &mockTransactionManager{})

	if _, err := uc.Execute(context.Background(), validInput()); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if joinedUserID == "" {
		t.Fatal("expected AddToGlobalGroup to be called with the new user's id")
	}
}

func TestCreateUser_GlobalGroupJoinError(t *testing.T) {
	repo := newNoConflictRepo()
	joiner := newNoopGlobalGroupJoiner()
	joiner.addToGlobalGroupFn = func(_ context.Context, _ string) error {
		return apperror.NewInternal()
	}
	uc := NewCreateUserUseCase(repo, joiner, &mockTransactionManager{})

	_, err := uc.Execute(context.Background(), validInput())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	appErr, ok := err.(*apperror.AppError)
	if !ok {
		t.Fatalf("expected *apperror.AppError, got %T", err)
	}
	if appErr.Code != apperror.ErrCodeInternalError {
		t.Errorf("expected code INTERNAL_ERROR, got %q", appErr.Code)
	}
}

func TestCreateUser_RepositorySaveError(t *testing.T) {
	repo := newNoConflictRepo()
	repo.saveFn = func(_ context.Context, _ *domain.User) error {
		return apperror.NewInternal()
	}
	uc := NewCreateUserUseCase(repo, newNoopGlobalGroupJoiner(), &mockTransactionManager{})

	_, err := uc.Execute(context.Background(), validInput())
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	appErr, ok := err.(*apperror.AppError)
	if !ok {
		t.Fatalf("expected *apperror.AppError, got %T", err)
	}
	if appErr.Code != apperror.ErrCodeInternalError {
		t.Errorf("expected code INTERNAL_ERROR, got %q", appErr.Code)
	}
}
