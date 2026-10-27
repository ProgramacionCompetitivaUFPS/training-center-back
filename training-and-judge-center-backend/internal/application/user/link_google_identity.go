package user

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	appshared "github.com/training-judge-center/backend/internal/application/shared"
	"github.com/training-judge-center/backend/internal/domain/user"
	"github.com/training-judge-center/backend/pkg/apperror"
	"github.com/training-judge-center/backend/pkg/emailtemplate"
)

type LinkGoogleIdentityInput struct {
	UserID  string
	IDToken string
}

type LinkGoogleIdentityUseCase struct {
	userRepo          user.Repository
	oauthIdentityRepo user.OAuthIdentityRepository
	googleVerifier    GoogleIDTokenVerifier
	emailSender       appshared.EmailSender
}

func NewLinkGoogleIdentityUseCase(userRepo user.Repository, oauthIdentityRepo user.OAuthIdentityRepository, googleVerifier GoogleIDTokenVerifier, emailSender appshared.EmailSender) *LinkGoogleIdentityUseCase {
	return &LinkGoogleIdentityUseCase{userRepo: userRepo, oauthIdentityRepo: oauthIdentityRepo, googleVerifier: googleVerifier, emailSender: emailSender}
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

	if err := uc.oauthIdentityRepo.Save(ctx, identity); err != nil {
		return err
	}

	if foundUser, findErr := uc.userRepo.FindByID(ctx, in.UserID); findErr == nil && foundUser != nil {
		_ = uc.emailSender.Send(ctx, appshared.EmailMessage{
			To:      foundUser.Email().String(),
			Subject: "Security Alert: Google Account Linked",
			Body:    fmt.Sprintf("The Google account %s has been linked to your account and can now be used to log in. If you did not make this change, unlink it from your account settings immediately or contact support.", claims.Email),
			HTMLBody: emailtemplate.Wrap("Security Alert: Google Account Linked",
				fmt.Sprintf("<p style=\"margin:0 0 12px;\">The Google account <strong>%s</strong> has been linked to your Training Center account and can now be used to log in.</p>"+
					"<p style=\"margin:0;color:#b91c1c;font-size:14px;\"><strong>If you did not make this change</strong>, unlink it from your account settings immediately or contact support.</p>", claims.Email)),
		})
	}

	return nil
}
