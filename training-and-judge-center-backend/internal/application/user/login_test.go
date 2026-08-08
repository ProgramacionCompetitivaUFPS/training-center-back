package user

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/training-judge-center/backend/internal/domain/shared"
	domain "github.com/training-judge-center/backend/internal/domain/user"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type mockTokenService struct {
	generateTokenFn func(ctx context.Context, user *domain.User) (string, error)
	validateTokenFn func(tokenString string) (*domain.TokenClaims, error)
}

func (m *mockTokenService) GenerateToken(ctx context.Context, user *domain.User) (string, error) {
	return m.generateTokenFn(ctx, user)
}

func (m *mockTokenService) ValidateToken(tokenString string) (*domain.TokenClaims, error) {
	return m.validateTokenFn(tokenString)
}

func newActiveUser() *domain.User {
	password, _ := domain.NewPassword("Secret1!")
	emailStr := "test@example.com"
	return domain.RestoreUser(
		"user-uuid-123",
		&emailStr,
		password.Hash(),
		"Test User",
		"testuser",
		"Colombia",
		"CÃºcuta",
		"UFPS",
		shared.RoleContestant.String(),
		domain.StatusActive.String(),
		time.Now(),
		nil,
		nil,
	)
}

func newLoginDeps() (*mockUserRepository, *mockTokenService) {
	repo := newNoConflictRepo()
	tokenService := &mockTokenService{
		generateTokenFn: func(_ context.Context, _ *domain.User) (string, error) { return "mock-jwt-token", nil },
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
	uc := NewLoginUseCase(repo, &mockRefreshTokenRepository{}, tokenSvc, &mockRateLimiter{})

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
	uc := NewLoginUseCase(repo, &mockRefreshTokenRepository{}, tokenSvc, &mockRateLimiter{})

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
	if appErr.Code != apperror.ErrCodeValidationError {
		t.Errorf("expected code VALIDATION_ERROR, got %q", appErr.Code)
	}
	if len(appErr.Details) != 2 {
		t.Errorf("expected 2 field errors (email + password), got %d", len(appErr.Details))
	}
}

func TestLogin_InvalidEmailFormat(t *testing.T) {
	repo, tokenSvc := newLoginDeps()
	uc := NewLoginUseCase(repo, &mockRefreshTokenRepository{}, tokenSvc, &mockRateLimiter{})

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
	if appErr.Code != ErrCodeInvalidCredentials {
		t.Errorf("expected code INVALID_CREDENTIALS, got %q", appErr.Code)
	}
	if appErr.Kind != apperror.KindUnauthorized {
		t.Errorf("expected kind UNAUTHORIZED, got %s", appErr.Kind)
	}
}

func TestLogin_UserNotFound(t *testing.T) {
	repo, tokenSvc := newLoginDeps()
	repo.findByEmailFn = func(_ context.Context, _ domain.Email) (*domain.User, error) {
		return nil, nil
	}
	uc := NewLoginUseCase(repo, &mockRefreshTokenRepository{}, tokenSvc, &mockRateLimiter{})

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
	if appErr.Code != ErrCodeInvalidCredentials {
		t.Errorf("expected code INVALID_CREDENTIALS, got %q", appErr.Code)
	}
}

func TestLogin_DeactivatedAccount(t *testing.T) {
	repo, tokenSvc := newLoginDeps()
	deactivatedUser := newActiveUser()
	deactivatedUser.Deactivate("test_suffix", time.Now())
	repo.findByEmailFn = func(_ context.Context, _ domain.Email) (*domain.User, error) {
		return deactivatedUser, nil
	}
	uc := NewLoginUseCase(repo, &mockRefreshTokenRepository{}, tokenSvc, &mockRateLimiter{})

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
	if appErr.Code != ErrCodeAccountDeactivated {
		t.Errorf("expected code ACCOUNT_DEACTIVATED, got %q", appErr.Code)
	}
	if appErr.Kind != apperror.KindForbidden {
		t.Errorf("expected kind FORBIDDEN, got %s", appErr.Kind)
	}
}

func TestLogin_WrongPassword(t *testing.T) {
	repo, tokenSvc := newLoginDeps()
	activeUser := newActiveUser()
	repo.findByEmailFn = func(_ context.Context, _ domain.Email) (*domain.User, error) {
		return activeUser, nil
	}
	uc := NewLoginUseCase(repo, &mockRefreshTokenRepository{}, tokenSvc, &mockRateLimiter{})

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
	if appErr.Code != ErrCodeInvalidCredentials {
		t.Errorf("expected code INVALID_CREDENTIALS, got %q", appErr.Code)
	}
}

func TestLogin_SameErrorForNotFoundAndWrongPassword(t *testing.T) {
	repo, tokenSvc := newLoginDeps()

	// Test user not found
	repo.findByEmailFn = func(_ context.Context, _ domain.Email) (*domain.User, error) {
		return nil, nil
	}
	uc := NewLoginUseCase(repo, &mockRefreshTokenRepository{}, tokenSvc, &mockRateLimiter{})
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
	tokenSvc.generateTokenFn = func(_ context.Context, _ *domain.User) (string, error) {
		return "", apperror.NewInternal()
	}
	uc := NewLoginUseCase(repo, &mockRefreshTokenRepository{}, tokenSvc, &mockRateLimiter{})

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
	if appErr.Code != apperror.ErrCodeInternalError {
		t.Errorf("expected code INTERNAL_ERROR, got %q", appErr.Code)
	}
}

func TestLogin_TokenGenerationError_DoesNotSaveRefreshToken(t *testing.T) {
	repo, tokenSvc := newLoginDeps()
	activeUser := newActiveUser()
	repo.findByEmailFn = func(_ context.Context, _ domain.Email) (*domain.User, error) {
		return activeUser, nil
	}
	tokenSvc.generateTokenFn = func(_ context.Context, _ *domain.User) (string, error) {
		return "", apperror.NewInternal()
	}
	saveCalled := false
	refreshRepo := &mockRefreshTokenRepository{
		saveFn: func(_ context.Context, _ *domain.RefreshToken) error {
			saveCalled = true
			return nil
		},
	}
	uc := NewLoginUseCase(repo, refreshRepo, tokenSvc, &mockRateLimiter{})

	_, err := uc.Execute(context.Background(), LoginInput{
		Email:    "test@example.com",
		Password: "Secret1!",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if saveCalled {
		t.Error("expected refreshTokenRepo.Save NOT to be called when GenerateToken fails — no orphaned refresh token")
	}
}

func TestLogin_RepositoryFindByEmailError(t *testing.T) {
	repo, tokenSvc := newLoginDeps()
	repo.findByEmailFn = func(_ context.Context, _ domain.Email) (*domain.User, error) {
		return nil, apperror.NewInternal()
	}
	uc := NewLoginUseCase(repo, &mockRefreshTokenRepository{}, tokenSvc, &mockRateLimiter{})

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
	if appErr.Code != apperror.ErrCodeInternalError {
		t.Errorf("expected code INTERNAL_ERROR, got %q", appErr.Code)
	}
}

func TestLogin_PasswordAtBcryptBoundary(t *testing.T) {
	pass72 := "A1!" + strings.Repeat("a", 69)
	password, err := domain.NewPassword(pass72)
	if err != nil {
		t.Fatalf("setup: %v", err)
	}
	emailStr := "boundary@example.com"
	u := domain.RestoreUser(
		"user-boundary",
		&emailStr,
		password.Hash(),
		"Boundary User",
		"boundaryuser",
		"Colombia",
		"CÃºcuta",
		"UFPS",
		shared.RoleContestant.String(),
		domain.StatusActive.String(),
		time.Now(),
		nil,
		nil,
	)

	repo, tokenSvc := newLoginDeps()
	repo.findByEmailFn = func(_ context.Context, _ domain.Email) (*domain.User, error) {
		return u, nil
	}
	uc := NewLoginUseCase(repo, &mockRefreshTokenRepository{}, tokenSvc, &mockRateLimiter{})

	_, err = uc.Execute(context.Background(), LoginInput{
		Email:    "boundary@example.com",
		Password: pass72,
	})
	if err != nil {
		t.Fatalf("login with 72-byte password must succeed, got: %v", err)
	}
}

func TestLogin_RateLimitExceeded(t *testing.T) {
	repo, tokenSvc := newLoginDeps()
	rl := &mockRateLimiter{
		allowFn: func(_ context.Context, _ string, _ int, _ time.Duration) (bool, error) {
			return false, nil
		},
	}
	uc := NewLoginUseCase(repo, &mockRefreshTokenRepository{}, tokenSvc, rl)

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
	if appErr.Code != ErrCodeTooManyRequests {
		t.Errorf("expected code TOO_MANY_REQUESTS, got %q", appErr.Code)
	}
	if appErr.Kind != apperror.KindTooManyRequests {
		t.Errorf("expected kind TOO_MANY_REQUESTS, got %s", appErr.Kind)
	}
}

func TestLogin_RateLimitError(t *testing.T) {
	repo, tokenSvc := newLoginDeps()
	rl := &mockRateLimiter{
		allowFn: func(_ context.Context, _ string, _ int, _ time.Duration) (bool, error) {
			return false, apperror.NewInternal()
		},
	}
	uc := NewLoginUseCase(repo, &mockRefreshTokenRepository{}, tokenSvc, rl)

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
	if appErr.Kind != apperror.KindInternal {
		t.Errorf("expected kind INTERNAL, got %s", appErr.Kind)
	}
}

func TestLogin_Success_ResetsRateLimit(t *testing.T) {
	repo, tokenSvc := newLoginDeps()
	activeUser := newActiveUser()
	repo.findByEmailFn = func(_ context.Context, _ domain.Email) (*domain.User, error) {
		return activeUser, nil
	}

	resetCalls := 0
	var resetKey string
	rl := &mockRateLimiter{
		resetFn: func(_ context.Context, key string) error {
			resetCalls++
			resetKey = key
			return nil
		},
	}
	uc := NewLoginUseCase(repo, &mockRefreshTokenRepository{}, tokenSvc, rl)

	_, err := uc.Execute(context.Background(), LoginInput{
		Email:    "test@example.com",
		Password: "Secret1!",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if resetCalls != 1 {
		t.Errorf("expected Reset to be called exactly once, got %d calls", resetCalls)
	}
	if resetKey != "rate_limit:login:test@example.com" {
		t.Errorf("expected reset key %q, got %q", "rate_limit:login:test@example.com", resetKey)
	}
}

func TestLogin_WrongPassword_DoesNotResetRateLimit(t *testing.T) {
	repo, tokenSvc := newLoginDeps()
	activeUser := newActiveUser()
	repo.findByEmailFn = func(_ context.Context, _ domain.Email) (*domain.User, error) {
		return activeUser, nil
	}

	resetCalled := false
	rl := &mockRateLimiter{
		resetFn: func(_ context.Context, _ string) error {
			resetCalled = true
			return nil
		},
	}
	uc := NewLoginUseCase(repo, &mockRefreshTokenRepository{}, tokenSvc, rl)

	_, err := uc.Execute(context.Background(), LoginInput{
		Email:    "test@example.com",
		Password: "WrongPass1!",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if resetCalled {
		t.Error("expected Reset not to be called on failed login")
	}
}

func TestLogin_Success_IssuesRefreshToken(t *testing.T) {
	repo, tokenSvc := newLoginDeps()
	activeUser := newActiveUser()
	repo.findByEmailFn = func(_ context.Context, _ domain.Email) (*domain.User, error) {
		return activeUser, nil
	}

	var savedToken *domain.RefreshToken
	refreshRepo := &mockRefreshTokenRepository{
		saveFn: func(_ context.Context, token *domain.RefreshToken) error {
			savedToken = token
			return nil
		},
	}
	uc := NewLoginUseCase(repo, refreshRepo, tokenSvc, &mockRateLimiter{})

	result, err := uc.Execute(context.Background(), LoginInput{
		Email:    "test@example.com",
		Password: "Secret1!",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.RefreshToken == "" {
		t.Error("expected a non-empty refresh token")
	}
	if result.SessionExpiresAt.IsZero() {
		t.Error("expected a non-zero sessionExpiresAt")
	}
	if savedToken == nil {
		t.Fatal("expected refreshTokenRepo.Save to be called")
	}
	if savedToken.UserID() != activeUser.ID() {
		t.Errorf("expected saved token userID %q, got %q", activeUser.ID(), savedToken.UserID())
	}
	if !savedToken.AbsoluteExpiresAt().Equal(result.SessionExpiresAt) {
		t.Errorf("expected saved token ceiling to match output SessionExpiresAt")
	}
}

func TestLogin_DefaultCeiling_UsesShortCeiling(t *testing.T) {
	repo, tokenSvc := newLoginDeps()
	activeUser := newActiveUser()
	repo.findByEmailFn = func(_ context.Context, _ domain.Email) (*domain.User, error) {
		return activeUser, nil
	}

	var savedToken *domain.RefreshToken
	refreshRepo := &mockRefreshTokenRepository{
		saveFn: func(_ context.Context, token *domain.RefreshToken) error {
			savedToken = token
			return nil
		},
	}
	uc := NewLoginUseCase(repo, refreshRepo, tokenSvc, &mockRateLimiter{})

	before := time.Now()
	_, err := uc.Execute(context.Background(), LoginInput{
		Email:    "test@example.com",
		Password: "Secret1!",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	want := before.Add(domain.DefaultSessionCeiling)
	diff := savedToken.AbsoluteExpiresAt().Sub(want)
	if diff < -time.Second || diff > time.Second {
		t.Errorf("expected ceiling near %v, got %v", want, savedToken.AbsoluteExpiresAt())
	}
}

func TestLogin_RememberSession_UsesLongCeiling(t *testing.T) {
	repo, tokenSvc := newLoginDeps()
	activeUser := newActiveUser()
	repo.findByEmailFn = func(_ context.Context, _ domain.Email) (*domain.User, error) {
		return activeUser, nil
	}

	var savedToken *domain.RefreshToken
	refreshRepo := &mockRefreshTokenRepository{
		saveFn: func(_ context.Context, token *domain.RefreshToken) error {
			savedToken = token
			return nil
		},
	}
	uc := NewLoginUseCase(repo, refreshRepo, tokenSvc, &mockRateLimiter{})

	before := time.Now()
	_, err := uc.Execute(context.Background(), LoginInput{
		Email:           "test@example.com",
		Password:        "Secret1!",
		RememberSession: true,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	want := before.Add(domain.RememberSessionCeiling)
	diff := savedToken.AbsoluteExpiresAt().Sub(want)
	if diff < -time.Second || diff > time.Second {
		t.Errorf("expected ceiling near %v, got %v", want, savedToken.AbsoluteExpiresAt())
	}
}

func TestLogin_RefreshTokenSaveError(t *testing.T) {
	repo, tokenSvc := newLoginDeps()
	activeUser := newActiveUser()
	repo.findByEmailFn = func(_ context.Context, _ domain.Email) (*domain.User, error) {
		return activeUser, nil
	}

	refreshRepo := &mockRefreshTokenRepository{
		saveFn: func(_ context.Context, _ *domain.RefreshToken) error {
			return apperror.NewInternal()
		},
	}
	uc := NewLoginUseCase(repo, refreshRepo, tokenSvc, &mockRateLimiter{})

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
	if appErr.Code != apperror.ErrCodeInternalError {
		t.Errorf("expected code INTERNAL_ERROR, got %q", appErr.Code)
	}
}
