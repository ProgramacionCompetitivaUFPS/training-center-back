package user

import (
	"context"

	appshared "github.com/training-judge-center/backend/internal/application/shared"
	"github.com/training-judge-center/backend/internal/domain/user"
	"github.com/training-judge-center/backend/pkg/apperror"
	"github.com/training-judge-center/backend/pkg/emailtemplate"
)

type UnlinkGoogleIdentityInput struct {
	UserID string
}

type UnlinkGoogleIdentityUseCase struct {
	userRepo          user.Repository
	oauthIdentityRepo user.OAuthIdentityRepository
	emailSender       appshared.EmailSender
}

func NewUnlinkGoogleIdentityUseCase(userRepo user.Repository, oauthIdentityRepo user.OAuthIdentityRepository, emailSender appshared.EmailSender) *UnlinkGoogleIdentityUseCase {
	return &UnlinkGoogleIdentityUseCase{userRepo: userRepo, oauthIdentityRepo: oauthIdentityRepo, emailSender: emailSender}
}

func (uc *UnlinkGoogleIdentityUseCase) Execute(ctx context.Context, in UnlinkGoogleIdentityInput) error {
	foundUser, err := uc.userRepo.FindByID(ctx, in.UserID)
	if err != nil {
		return err
	}
	if foundUser == nil {
		return apperror.NewNotFound(user.ErrCodeUserNotFound, "User not found")
	}
	if !foundUser.Password().HasPassword() {
		return apperror.NewConflict(ErrCodeCannotUnlinkLastCredential, "cannot unlink Google account without a password set; set a password first")
	}

	identity, err := uc.oauthIdentityRepo.FindByUserID(ctx, in.UserID, user.OAuthProviderGoogle)
	if err != nil {
		return err
	}
	if identity == nil {
		return apperror.NewNotFound(user.ErrCodeOAuthIdentityNotFound, "no linked Google account found for this user")
	}

	if err := uc.oauthIdentityRepo.DeleteByUserID(ctx, in.UserID, user.OAuthProviderGoogle); err != nil {
		return err
	}

	_ = uc.emailSender.Send(ctx, appshared.EmailMessage{
		To:      foundUser.Email().String(),
		Subject: "Security Alert: Google Account Unlinked",
		Body:    "Your Google account has been unlinked from your account and can no longer be used to log in. If you did not make this change, please review your account security immediately.",
		HTMLBody: emailtemplate.Wrap("Security Alert: Google Account Unlinked",
			"<p style=\"margin:0 0 12px;\">Your Google account has been unlinked from your Training Center account and can no longer be used to log in.</p>"+
				"<p style=\"margin:0;color:#b91c1c;font-size:14px;\"><strong>If you did not make this change</strong>, please review your account security immediately.</p>"),
	})

	return nil
}
