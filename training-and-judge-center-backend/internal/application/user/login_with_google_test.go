package user

import (
	"context"
	"testing"
	"time"

	"github.com/training-judge-center/backend/internal/domain/shared"
	domain "github.com/training-judge-center/backend/internal/domain/user"
	"github.com/training-judge-center/backend/pkg/apperror"
)

func newGoogleLoginDeps() (*mockUserRepository, *mockOAuthIdentityRepository, *mockTokenService, *mockGoogleIDTokenVerifier) {
	repo, tokenSvc := newLoginDeps()
	oauthRepo := &mockOAuthIdentityRepository{}
	googleVerifier := &mockGoogleIDTokenVerifier{}
	return repo, oauthRepo, tokenSvc, googleVerifier
}

func newGoogleLoginUseCase(
	repo *mockUserRepository,
	oauthRepo *mockOAuthIdentityRepository,
	tokenSvc *mockTokenService,
	googleVerifier *mockGoogleIDTokenVerifier,
) *LoginWithGoogleUseCase {
	return NewLoginWithGoogleUseCase(repo, oauthRepo, &mockRefreshTokenRepository{}, tokenSvc, &mockRefreshTokenCodec{}, googleVerifier, &mockTransactionManager{})
}

func TestLoginWithGoogle_ExistingIdentity_IssuesSession(t *testing.T) {
	repo, oauthRepo, tokenSvc, googleVerifier := newGoogleLoginDeps()
	activeUser := newActiveUser()

	oauthRepo.findByProviderFn = func(_ context.Context, provider domain.OAuthProvider, providerUserID string) (*domain.OAuthIdentity, error) {
		identity, _ := domain.NewOAuthIdentity("identity-1", activeUser.ID(), domain.OAuthProviderGoogle, "google-sub-123", time.Now())
		return identity, nil
	}
	repo.findByIDFn = func(_ context.Context, id string) (*domain.User, error) {
		if id == activeUser.ID() {
			return activeUser, nil
		}
		return nil, nil
	}

	saveIdentityCalled := false
	oauthRepo.saveFn = func(_ context.Context, _ *domain.OAuthIdentity) error {
		saveIdentityCalled = true
		return nil
	}

	uc := newGoogleLoginUseCase(repo, oauthRepo, tokenSvc, googleVerifier)

	result, err := uc.Execute(context.Background(), LoginWithGoogleInput{IDToken: "valid-token"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.Token != "mock-jwt-token" {
		t.Errorf("expected token %q, got %q", "mock-jwt-token", result.Token)
	}
	if result.User.ID != activeUser.ID() {
		t.Errorf("expected user ID %q, got %q", activeUser.ID(), result.User.ID)
	}
	if saveIdentityCalled {
		t.Error("expected oauthIdentityRepo.Save NOT to be called when the identity already exists")
	}
}

func TestLoginWithGoogle_ExistingEmailNoIdentity_LinksAccount(t *testing.T) {
	repo, oauthRepo, tokenSvc, googleVerifier := newGoogleLoginDeps()
	activeUser := newActiveUser()

	oauthRepo.findByProviderFn = func(_ context.Context, _ domain.OAuthProvider, _ string) (*domain.OAuthIdentity, error) {
		return nil, nil
	}
	repo.findByEmailFn = func(_ context.Context, _ domain.Email) (*domain.User, error) {
		return activeUser, nil
	}

	var savedIdentity *domain.OAuthIdentity
	oauthRepo.saveFn = func(_ context.Context, identity *domain.OAuthIdentity) error {
		savedIdentity = identity
		return nil
	}
	userSaveCalled := false
	repo.saveFn = func(_ context.Context, _ *domain.User) error {
		userSaveCalled = true
		return nil
	}

	googleVerifier.verifyFn = func(_ context.Context, _ string) (*GoogleClaims, error) {
		return &GoogleClaims{Sub: "google-sub-123", Email: "test@example.com", EmailVerified: true, Name: "Test User"}, nil
	}

	uc := newGoogleLoginUseCase(repo, oauthRepo, tokenSvc, googleVerifier)

	result, err := uc.Execute(context.Background(), LoginWithGoogleInput{IDToken: "valid-token"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.User.ID != activeUser.ID() {
		t.Errorf("expected user ID %q, got %q", activeUser.ID(), result.User.ID)
	}
	if userSaveCalled {
		t.Error("expected userRepo.Save NOT to be called for an already-existing account")
	}
	if savedIdentity == nil {
		t.Fatal("expected oauthIdentityRepo.Save to be called to link the account")
	}
	if savedIdentity.UserID() != activeUser.ID() {
		t.Errorf("expected linked identity userID %q, got %q", activeUser.ID(), savedIdentity.UserID())
	}
	if savedIdentity.ProviderUserID() != "google-sub-123" {
		t.Errorf("expected linked identity providerUserID %q, got %q", "google-sub-123", savedIdentity.ProviderUserID())
	}
}

func TestLoginWithGoogle_NewEmail_CreatesUserAsContestant(t *testing.T) {
	repo, oauthRepo, tokenSvc, googleVerifier := newGoogleLoginDeps()

	oauthRepo.findByProviderFn = func(_ context.Context, _ domain.OAuthProvider, _ string) (*domain.OAuthIdentity, error) {
		return nil, nil
	}
	repo.findByEmailFn = func(_ context.Context, _ domain.Email) (*domain.User, error) {
		return nil, nil
	}

	var savedUser *domain.User
	repo.saveFn = func(_ context.Context, u *domain.User) error {
		savedUser = u
		return nil
	}
	identitySaved := false
	oauthRepo.saveFn = func(_ context.Context, _ *domain.OAuthIdentity) error {
		identitySaved = true
		return nil
	}

	googleVerifier.verifyFn = func(_ context.Context, _ string) (*GoogleClaims, error) {
		return &GoogleClaims{Sub: "google-sub-new", Email: "brandnew@example.com", EmailVerified: true, Name: "Brand New"}, nil
	}

	uc := newGoogleLoginUseCase(repo, oauthRepo, tokenSvc, googleVerifier)

	result, err := uc.Execute(context.Background(), LoginWithGoogleInput{IDToken: "valid-token"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if savedUser == nil {
		t.Fatal("expected userRepo.Save to be called for a brand-new account")
	}
	if savedUser.Role() != shared.RoleContestant {
		t.Errorf("expected new user role %q, got %q", shared.RoleContestant, savedUser.Role())
	}
	if savedUser.Password().HasPassword() {
		t.Error("expected new Google-only user to have no local password")
	}
	if !identitySaved {
		t.Error("expected oauthIdentityRepo.Save to be called for the new identity")
	}
	if result.User.ID != savedUser.ID() {
		t.Errorf("expected output user ID %q, got %q", savedUser.ID(), result.User.ID)
	}
}

func TestLoginWithGoogle_InvalidToken_ReturnsUnauthorized(t *testing.T) {
	repo, oauthRepo, tokenSvc, googleVerifier := newGoogleLoginDeps()
	googleVerifier.verifyFn = func(_ context.Context, _ string) (*GoogleClaims, error) {
		return nil, apperror.NewUnauthorized("IGNORED", "google rejected the token")
	}
	uc := newGoogleLoginUseCase(repo, oauthRepo, tokenSvc, googleVerifier)

	_, err := uc.Execute(context.Background(), LoginWithGoogleInput{IDToken: "bad-token"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	appErr, ok := err.(*apperror.AppError)
	if !ok {
		t.Fatalf("expected *apperror.AppError, got %T", err)
	}
	if appErr.Code != ErrCodeInvalidGoogleToken {
		t.Errorf("expected code %q, got %q", ErrCodeInvalidGoogleToken, appErr.Code)
	}
	if appErr.Kind != apperror.KindUnauthorized {
		t.Errorf("expected kind UNAUTHORIZED, got %s", appErr.Kind)
	}
}

func TestLoginWithGoogle_EmailNotVerified_ReturnsUnauthorized(t *testing.T) {
	repo, oauthRepo, tokenSvc, googleVerifier := newGoogleLoginDeps()
	googleVerifier.verifyFn = func(_ context.Context, _ string) (*GoogleClaims, error) {
		return &GoogleClaims{Sub: "google-sub-123", Email: "unverified@example.com", EmailVerified: false}, nil
	}
	uc := newGoogleLoginUseCase(repo, oauthRepo, tokenSvc, googleVerifier)

	_, err := uc.Execute(context.Background(), LoginWithGoogleInput{IDToken: "valid-token"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	appErr, ok := err.(*apperror.AppError)
	if !ok {
		t.Fatalf("expected *apperror.AppError, got %T", err)
	}
	if appErr.Code != ErrCodeGoogleEmailNotVerified {
		t.Errorf("expected code %q, got %q", ErrCodeGoogleEmailNotVerified, appErr.Code)
	}
}

func TestLoginWithGoogle_DeactivatedAccount_ReturnsForbidden(t *testing.T) {
	repo, oauthRepo, tokenSvc, googleVerifier := newGoogleLoginDeps()
	deactivatedUser := newActiveUser()
	deactivatedUser.Deactivate("test_suffix", time.Now())

	oauthRepo.findByProviderFn = func(_ context.Context, _ domain.OAuthProvider, _ string) (*domain.OAuthIdentity, error) {
		identity, _ := domain.NewOAuthIdentity("identity-1", deactivatedUser.ID(), domain.OAuthProviderGoogle, "google-sub-123", time.Now())
		return identity, nil
	}
	repo.findByIDFn = func(_ context.Context, _ string) (*domain.User, error) {
		return deactivatedUser, nil
	}

	uc := newGoogleLoginUseCase(repo, oauthRepo, tokenSvc, googleVerifier)

	_, err := uc.Execute(context.Background(), LoginWithGoogleInput{IDToken: "valid-token"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	appErr, ok := err.(*apperror.AppError)
	if !ok {
		t.Fatalf("expected *apperror.AppError, got %T", err)
	}
	if appErr.Code != ErrCodeAccountDeactivated {
		t.Errorf("expected code %q, got %q", ErrCodeAccountDeactivated, appErr.Code)
	}
	if appErr.Kind != apperror.KindForbidden {
		t.Errorf("expected kind FORBIDDEN, got %s", appErr.Kind)
	}
}

func TestLoginWithGoogle_OrphanedIdentity_ReturnsInternalNotPanic(t *testing.T) {
	repo, oauthRepo, tokenSvc, googleVerifier := newGoogleLoginDeps()

	oauthRepo.findByProviderFn = func(_ context.Context, _ domain.OAuthProvider, _ string) (*domain.OAuthIdentity, error) {
		identity, _ := domain.NewOAuthIdentity("identity-1", "deleted-user-id", domain.OAuthProviderGoogle, "google-sub-123", time.Now())
		return identity, nil
	}
	repo.findByIDFn = func(_ context.Context, _ string) (*domain.User, error) {
		return nil, nil // user_id no longer resolves to a user
	}

	uc := newGoogleLoginUseCase(repo, oauthRepo, tokenSvc, googleVerifier)

	_, err := uc.Execute(context.Background(), LoginWithGoogleInput{IDToken: "valid-token"})
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

func TestLoginWithGoogle_ConcurrentSignup_LosesRaceAndRecovers(t *testing.T) {
	repo, oauthRepo, tokenSvc, googleVerifier := newGoogleLoginDeps()

	oauthRepo.findByProviderFn = func(_ context.Context, _ domain.OAuthProvider, _ string) (*domain.OAuthIdentity, error) {
		return nil, nil // no identity yet, on both attempts
	}
	repo.findByEmailFn = func(_ context.Context, _ domain.Email) (*domain.User, error) {
		return nil, nil // this request thinks the account doesn't exist either
	}

	txManager := &mockTransactionManager{
		withTxFn: func(_ context.Context, fn func(txCtx context.Context) error) error {
			return apperror.NewConflict(domain.ErrCodeEmailConflict, "email already in use") // lost the race
		},
	}

	winner := newActiveUser()
	var findByProviderCalls int
	oauthRepo.findByProviderFn = func(_ context.Context, _ domain.OAuthProvider, _ string) (*domain.OAuthIdentity, error) {
		findByProviderCalls++
		if findByProviderCalls == 1 {
			return nil, nil // first check: nothing yet
		}
		identity, _ := domain.NewOAuthIdentity("identity-1", winner.ID(), domain.OAuthProviderGoogle, "google-sub-123", time.Now())
		return identity, nil // post-conflict re-check: the winner's identity
	}
	repo.findByIDFn = func(_ context.Context, id string) (*domain.User, error) {
		if id == winner.ID() {
			return winner, nil
		}
		return nil, nil
	}

	googleVerifier.verifyFn = func(_ context.Context, _ string) (*GoogleClaims, error) {
		return &GoogleClaims{Sub: "google-sub-123", Email: "test@example.com", EmailVerified: true, Name: "Test User"}, nil
	}

	uc := NewLoginWithGoogleUseCase(repo, oauthRepo, &mockRefreshTokenRepository{}, tokenSvc, &mockRefreshTokenCodec{}, googleVerifier, txManager)

	result, err := uc.Execute(context.Background(), LoginWithGoogleInput{IDToken: "valid-token"})
	if err != nil {
		t.Fatalf("expected no error after recovering from lost race, got %v", err)
	}
	if result.User.ID != winner.ID() {
		t.Errorf("expected recovered user ID %q, got %q", winner.ID(), result.User.ID)
	}
	if findByProviderCalls != 2 {
		t.Errorf("expected FindByProvider to be called twice (initial + post-conflict recheck), got %d", findByProviderCalls)
	}
}

func TestLoginWithGoogle_MissingIDToken_ReturnsValidation(t *testing.T) {
	repo, oauthRepo, tokenSvc, googleVerifier := newGoogleLoginDeps()
	uc := newGoogleLoginUseCase(repo, oauthRepo, tokenSvc, googleVerifier)

	_, err := uc.Execute(context.Background(), LoginWithGoogleInput{IDToken: ""})
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
}
