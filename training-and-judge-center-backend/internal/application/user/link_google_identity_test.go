package user

import (
	"context"
	"testing"

	appshared "github.com/training-judge-center/backend/internal/application/shared"
	domain "github.com/training-judge-center/backend/internal/domain/user"
	"github.com/training-judge-center/backend/pkg/apperror"
)

func TestLinkGoogleIdentity_ValidToken_SavesIdentityForCurrentUser(t *testing.T) {
	oauthRepo := &mockOAuthIdentityRepository{}
	googleVerifier := &mockGoogleIDTokenVerifier{
		verifyFn: func(_ context.Context, _ string) (*GoogleClaims, error) {
			return &GoogleClaims{Sub: "google-sub-999", Email: "linked@example.com", EmailVerified: true, Name: "Linked User"}, nil
		},
	}

	var savedIdentity *domain.OAuthIdentity
	oauthRepo.saveFn = func(_ context.Context, identity *domain.OAuthIdentity) error {
		savedIdentity = identity
		return nil
	}

	uc := NewLinkGoogleIdentityUseCase(&mockUserRepository{}, oauthRepo, googleVerifier, &mockEmailSender{})

	err := uc.Execute(context.Background(), LinkGoogleIdentityInput{UserID: "user-abc", IDToken: "valid-token"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if savedIdentity == nil {
		t.Fatal("expected oauthIdentityRepo.Save to be called")
	}
	if savedIdentity.UserID() != "user-abc" {
		t.Errorf("expected linked identity userID %q, got %q", "user-abc", savedIdentity.UserID())
	}
	if savedIdentity.ProviderUserID() != "google-sub-999" {
		t.Errorf("expected linked identity providerUserID %q, got %q", "google-sub-999", savedIdentity.ProviderUserID())
	}
	if savedIdentity.Provider() != domain.OAuthProviderGoogle {
		t.Errorf("expected provider %q, got %q", domain.OAuthProviderGoogle, savedIdentity.Provider())
	}
}

func TestLinkGoogleIdentity_ValidToken_SendsSecurityAlertEmail(t *testing.T) {
	oauthRepo := &mockOAuthIdentityRepository{}
	userRepo := &mockUserRepository{
		findByIDFn: func(_ context.Context, _ string) (*domain.User, error) {
			return newActiveUser(), nil
		},
	}
	googleVerifier := &mockGoogleIDTokenVerifier{
		verifyFn: func(_ context.Context, _ string) (*GoogleClaims, error) {
			return &GoogleClaims{Sub: "google-sub-999", Email: "linked@example.com", EmailVerified: true}, nil
		},
	}

	var sentTo, sentSubject string
	emailSender := &mockEmailSender{
		sendFn: func(_ context.Context, msg appshared.EmailMessage) error {
			sentTo = msg.To
			sentSubject = msg.Subject
			return nil
		},
	}

	uc := NewLinkGoogleIdentityUseCase(userRepo, oauthRepo, googleVerifier, emailSender)

	err := uc.Execute(context.Background(), LinkGoogleIdentityInput{UserID: "user-uuid-123", IDToken: "valid-token"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if sentTo != "test@example.com" {
		t.Errorf("expected notification email to %q, got %q", "test@example.com", sentTo)
	}
	if sentSubject != "Security Alert: Google Account Linked" {
		t.Errorf("expected subject %q, got %q", "Security Alert: Google Account Linked", sentSubject)
	}
}

func TestLinkGoogleIdentity_MissingIDToken_ReturnsValidation(t *testing.T) {
	uc := NewLinkGoogleIdentityUseCase(&mockUserRepository{}, &mockOAuthIdentityRepository{}, &mockGoogleIDTokenVerifier{}, &mockEmailSender{})

	err := uc.Execute(context.Background(), LinkGoogleIdentityInput{UserID: "user-abc", IDToken: ""})
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

func TestLinkGoogleIdentity_InvalidToken_ReturnsUnauthorized(t *testing.T) {
	googleVerifier := &mockGoogleIDTokenVerifier{
		verifyFn: func(_ context.Context, _ string) (*GoogleClaims, error) {
			return nil, apperror.NewUnauthorized("IGNORED", "google rejected the token")
		},
	}
	uc := NewLinkGoogleIdentityUseCase(&mockUserRepository{}, &mockOAuthIdentityRepository{}, googleVerifier, &mockEmailSender{})

	err := uc.Execute(context.Background(), LinkGoogleIdentityInput{UserID: "user-abc", IDToken: "bad-token"})
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

func TestLinkGoogleIdentity_EmailNotVerified_ReturnsUnauthorized(t *testing.T) {
	googleVerifier := &mockGoogleIDTokenVerifier{
		verifyFn: func(_ context.Context, _ string) (*GoogleClaims, error) {
			return &GoogleClaims{Sub: "google-sub-999", Email: "unverified@example.com", EmailVerified: false}, nil
		},
	}
	uc := NewLinkGoogleIdentityUseCase(&mockUserRepository{}, &mockOAuthIdentityRepository{}, googleVerifier, &mockEmailSender{})

	err := uc.Execute(context.Background(), LinkGoogleIdentityInput{UserID: "user-abc", IDToken: "valid-token"})
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

func TestLinkGoogleIdentity_AlreadyLinkedElsewhere_PropagatesConflict(t *testing.T) {
	googleVerifier := &mockGoogleIDTokenVerifier{
		verifyFn: func(_ context.Context, _ string) (*GoogleClaims, error) {
			return &GoogleClaims{Sub: "google-sub-999", Email: "linked@example.com", EmailVerified: true}, nil
		},
	}
	oauthRepo := &mockOAuthIdentityRepository{
		saveFn: func(_ context.Context, _ *domain.OAuthIdentity) error {
			return apperror.NewConflict(domain.ErrCodeOAuthIdentityConflict, "this Google account is already linked to a user")
		},
	}
	uc := NewLinkGoogleIdentityUseCase(&mockUserRepository{}, oauthRepo, googleVerifier, &mockEmailSender{})

	err := uc.Execute(context.Background(), LinkGoogleIdentityInput{UserID: "user-abc", IDToken: "valid-token"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	appErr, ok := err.(*apperror.AppError)
	if !ok {
		t.Fatalf("expected *apperror.AppError, got %T", err)
	}
	if appErr.Code != domain.ErrCodeOAuthIdentityConflict {
		t.Errorf("expected code %q, got %q", domain.ErrCodeOAuthIdentityConflict, appErr.Code)
	}
}
