package user

import (
	"context"
	"log/slog"
	"time"

	appshared "github.com/training-judge-center/backend/internal/application/shared"
	"github.com/training-judge-center/backend/internal/domain/user"
	"github.com/training-judge-center/backend/pkg/apperror"
	"github.com/training-judge-center/backend/pkg/emailtemplate"
)

type SetPasswordInput struct {
	UserID      string
	NewPassword string
}

type SetPasswordUseCase struct {
	repo        user.Repository
	emailSender appshared.EmailSender
}

func NewSetPasswordUseCase(repo user.Repository, emailSender appshared.EmailSender) *SetPasswordUseCase {
	return &SetPasswordUseCase{repo: repo, emailSender: emailSender}
}

func (uc *SetPasswordUseCase) Execute(ctx context.Context, in SetPasswordInput) error {
	foundUser, err := uc.repo.FindByID(ctx, in.UserID)
	if err != nil {
		return err
	}
	if foundUser == nil {
		return apperror.NewNotFound(user.ErrCodeUserNotFound, "User not found")
	}

	if foundUser.Password().HasPassword() {
		return apperror.NewConflict(ErrCodePasswordAlreadySet, "this account already has a password; use the change-password flow instead")
	}

	newPassword, err := user.NewPassword(in.NewPassword)
	if err != nil {
		return apperror.NewValidation([]apperror.FieldError{
			{Field: "newPassword", Message: err.Error()},
		})
	}

	now := time.Now()
	if err := foundUser.UpdatePassword(newPassword, now); err != nil {
		slog.ErrorContext(ctx, "failed to set password on user domain object", "user_id", foundUser.ID(), "error", err)
		return apperror.NewInternal()
	}

	if err := uc.repo.Update(ctx, foundUser); err != nil {
		return err
	}

	_ = uc.emailSender.Send(ctx, appshared.EmailMessage{
		To:      foundUser.Email().String(),
		Subject: "Security Alert: Password Set",
		Body:    "A password has been set on your account and can now be used to log in. If you did not make this change, please review your account security immediately.",
		HTMLBody: emailtemplate.Wrap("Security Alert: Password Set",
			"<p style=\"margin:0 0 12px;\">A password has been set on your Training Center account and can now be used to log in.</p>"+
				"<p style=\"margin:0;color:#b91c1c;font-size:14px;\"><strong>If you did not make this change</strong>, please review your account security immediately.</p>"),
	})

	return nil
}
