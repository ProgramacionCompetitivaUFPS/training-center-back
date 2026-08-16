package user

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/training-judge-center/backend/internal/domain/user"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type LinkGoogleIdentityInput struct {
	UserID  string
	IDToken string
}

type LinkGoogleIdentityUseCase struct {
	oauthIdentityRepo user.OAuthIdentityRepository
	googleVerifier    GoogleIDTokenVerifier
}

func NewLinkGoogleIdentityUseCase(oauthIdentityRepo user.OAuthIdentityRepository, googleVerifier GoogleIDTokenVerifier) *LinkGoogleIdentityUseCase {
	return &LinkGoogleIdentityUseCase{oauthIdentityRepo: oauthIdentityRepo, googleVerifier: googleVerifier}
}

func (uc *LinkGoogleIdentityUseCase) Execute(ctx context.Context, in LinkGoogleIdentityInput) error {
	if in.IDToken == "" {
		return apperror.NewValidation([]apperror.FieldError{
			{Field: "id_token", Message: "id_token is required"},
		})
	}

	claims, err := uc.googleVerifier.Verify(ctx, in.IDToken)
	if err != nil {
		return apperror.NewUnauthorized(ErrCodeInvalidGoogleToken, "invalid Google ID token")
	}
	if !claims.EmailVerified {
		return apperror.NewUnauthorized(ErrCodeGoogleEmailNotVerified, "Google account email is not verified")
	}

	identity, err := user.NewOAuthIdentity(uuid.New().String(), in.UserID, user.OAuthProviderGoogle, claims.Sub, time.Now())
	if err != nil {
		return err
	}

	return uc.oauthIdentityRepo.Save(ctx, identity)
}
