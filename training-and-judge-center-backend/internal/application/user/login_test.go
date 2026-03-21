package user

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	domain "github.com/training-judge-center/backend/internal/domain/user"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type mockTokenService struct {
	generateTokenFn func(user *domain.User) (string, error)
	validateTokenFn func(tokenString string) (*domain.TokenClaims, error)
}

func (m *mockTokenService) GenerateToken(user *domain.User) (string, error) {
	return m.generateTokenFn(user)
}

func (m *mockTokenService) ValidateToken(tokenString string) (*domain.TokenClaims, error) {
	return m.validateTokenFn(tokenString)
}

func newActiveUser() *domain.User {
	email, _ := domain.NewEmail("test@example.com")
	password, _ := domain.NewPassword("Secret1!")
	nickname, _ := domain.NewNickname("testuser")

	return &domain.User{
		ID:          "user-uuid-123",
		Email:       &email,
		Password:    password,
		Name:        "Test User",
		Nickname:    nickname,
		Country:     "Colombia",
		City:        "Cúcuta",
		Institution: "UFPS",
		Role:        domain.RoleContestant,
		Status:      domain.StatusActive,
		CreatedAt:   time.Now(),
	}
}

func newLoginDeps() (*mockUserRepository, *mockTokenService) {
	repo := newNoConflictRepo()
	tokenService := &mockTokenService{
		generateTokenFn: func(_ *domain.User) (string, error) { return "mock-jwt-token", nil },
		validateTokenFn: func(_ string) (*domain.TokenClaims, error) { return nil, nil },
	}
	return repo, tokenService
}

func TestLogin_Success(t *testing.T) {
	repo, tokenSvc := newLoginDeps()
	activeUser := newActiveUser()
	repo.findByEmailFn = func(_ context.Context, _ domain.Email) (*domain.User, error) {
		return activeUser, nil
	}
	uc := NewLoginUseCase(repo, tokenSvc)

	result, err := uc.Execute(context.Background(), LoginInput{
		Email:    "test@example.com",
		Password: "Secret1!",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.Token != "mock-jwt-token" {
		t.Errorf("expected token %q, got %q", "mock-jwt-token", result.Token)
	}
	if result.User.ID != "user-uuid-123" {
		t.Errorf("expected user ID %q, got %q", "user-uuid-123", result.User.ID)
	}
}

func TestLogin_MissingFields(t *testing.T) {
	repo, tokenSvc := newLoginDeps()
	uc := NewLoginUseCase(repo, tokenSvc)

	_, err := uc.Execute(context.Background(), LoginInput{
		Email:    "",
		Password: "",
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
	if len(appErr.Details) != 2 {
		t.Errorf("expected 2 field errors (email + password), got %d", len(appErr.Details))
	}
}

func TestLogin_InvalidEmailFormat(t *testing.T) {
	repo, tokenSvc := newLoginDeps()
	uc := NewLoginUseCase(repo, tokenSvc)

	_, err := uc.Execute(context.Background(), LoginInput{
		Email:    "not-an-email",
		Password: "Secret1!",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	appErr, ok := err.(*apperror.AppError)
	if !ok {
		t.Fatalf("expected *apperror.AppError, got %T", err)
	}
	if appErr.Code != "INVALID_CREDENTIALS" {
		t.Errorf("expected code INVALID_CREDENTIALS, got %q", appErr.Code)
	}
	if appErr.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", appErr.StatusCode)
	}
}

func TestLogin_UserNotFound(t *testing.T) {
	repo, tokenSvc := newLoginDeps()
	repo.findByEmailFn = func(_ context.Context, _ domain.Email) (*domain.User, error) {
		return nil, nil
	}
	uc := NewLoginUseCase(repo, tokenSvc)

	_, err := uc.Execute(context.Background(), LoginInput{
		Email:    "nobody@example.com",
		Password: "Secret1!",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	appErr, ok := err.(*apperror.AppError)
	if !ok {
		t.Fatalf("expected *apperror.AppError, got %T", err)
	}
	if appErr.Code != "INVALID_CREDENTIALS" {
		t.Errorf("expected code INVALID_CREDENTIALS, got %q", appErr.Code)
	}
}

func TestLogin_DeactivatedAccount(t *testing.T) {
	repo, tokenSvc := newLoginDeps()
	deactivatedUser := newActiveUser()
	deactivatedUser.Status = domain.StatusDeactivated
	repo.findByEmailFn = func(_ context.Context, _ domain.Email) (*domain.User, error) {
		return deactivatedUser, nil
	}
	uc := NewLoginUseCase(repo, tokenSvc)

	_, err := uc.Execute(context.Background(), LoginInput{
		Email:    "test@example.com",
		Password: "Secret1!",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	appErr, ok := err.(*apperror.AppError)
	if !ok {
		t.Fatalf("expected *apperror.AppError, got %T", err)
	}
	if appErr.Code != "ACCOUNT_DEACTIVATED" {
		t.Errorf("expected code ACCOUNT_DEACTIVATED, got %q", appErr.Code)
	}
	if appErr.StatusCode != http.StatusForbidden {
		t.Errorf("expected status 403, got %d", appErr.StatusCode)
	}
}

func TestLogin_WrongPassword(t *testing.T) {
	repo, tokenSvc := newLoginDeps()
	activeUser := newActiveUser()
	repo.findByEmailFn = func(_ context.Context, _ domain.Email) (*domain.User, error) {
		return activeUser, nil
	}
	uc := NewLoginUseCase(repo, tokenSvc)

	_, err := uc.Execute(context.Background(), LoginInput{
		Email:    "test@example.com",
		Password: "WrongPass1!",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	appErr, ok := err.(*apperror.AppError)
	if !ok {
		t.Fatalf("expected *apperror.AppError, got %T", err)
	}
	if appErr.Code != "INVALID_CREDENTIALS" {
		t.Errorf("expected code INVALID_CREDENTIALS, got %q", appErr.Code)
	}
}

func TestLogin_SameErrorForNotFoundAndWrongPassword(t *testing.T) {
	repo, tokenSvc := newLoginDeps()

	// Test user not found
	repo.findByEmailFn = func(_ context.Context, _ domain.Email) (*domain.User, error) {
		return nil, nil
	}
	uc := NewLoginUseCase(repo, tokenSvc)
	_, errNotFound := uc.Execute(context.Background(), LoginInput{
		Email: "nobody@example.com", Password: "Secret1!",
	})

	// Test wrong password
	activeUser := newActiveUser()
	repo.findByEmailFn = func(_ context.Context, _ domain.Email) (*domain.User, error) {
		return activeUser, nil
	}
	_, errWrongPass := uc.Execute(context.Background(), LoginInput{
		Email: "test@example.com", Password: "WrongPass1!",
	})

	notFoundErr := errNotFound.(*apperror.AppError)
	wrongPassErr := errWrongPass.(*apperror.AppError)

	if notFoundErr.Code != wrongPassErr.Code {
		t.Errorf("error codes should be identical: %q vs %q", notFoundErr.Code, wrongPassErr.Code)
	}
	if notFoundErr.Message != wrongPassErr.Message {
		t.Errorf("error messages should be identical: %q vs %q", notFoundErr.Message, wrongPassErr.Message)
	}
}

func TestLogin_TokenGenerationError(t *testing.T) {
	repo, tokenSvc := newLoginDeps()
	activeUser := newActiveUser()
	repo.findByEmailFn = func(_ context.Context, _ domain.Email) (*domain.User, error) {
		return activeUser, nil
	}
	tokenSvc.generateTokenFn = func(_ *domain.User) (string, error) {
		return "", errors.New("signing key error")
	}
	uc := NewLoginUseCase(repo, tokenSvc)

	_, err := uc.Execute(context.Background(), LoginInput{
		Email:    "test@example.com",
		Password: "Secret1!",
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

func TestLogin_RepositoryFindByEmailError(t *testing.T) {
	repo, tokenSvc := newLoginDeps()
	repo.findByEmailFn = func(_ context.Context, _ domain.Email) (*domain.User, error) {
		return nil, errors.New("db connection lost")
	}
	uc := NewLoginUseCase(repo, tokenSvc)

	_, err := uc.Execute(context.Background(), LoginInput{
		Email:    "test@example.com",
		Password: "Secret1!",
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
