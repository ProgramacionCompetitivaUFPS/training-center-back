package user

import (
	"context"
	"log/slog"
	"time"

	"github.com/training-judge-center/backend/internal/domain/notification"
	"github.com/training-judge-center/backend/internal/domain/user"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type UpdatePasswordInput struct {
	UserID          string
	CurrentPassword string
	NewPassword     string
}

type UpdatePasswordUseCase struct {
	repo               user.UserRepository
	emailSender        notification.EmailSender
	sessionInvalidator user.SessionInvalidator
}

func NewUpdatePasswordUseCase(repo user.UserRepository, email notification.EmailSender, sessionInvalidator user.SessionInvalidator) *UpdatePasswordUseCase {
	return &UpdatePasswordUseCase{
		repo:               repo,
		emailSender:        email,
		sessionInvalidator: sessionInvalidator,
	}
}

func (uc *UpdatePasswordUseCase) Execute(ctx context.Context, input UpdatePasswordInput) error {
	foundUser, err := uc.repo.FindByID(ctx, input.UserID)
	if err != nil {
		return apperror.NewInternal()
	}
	if foundUser == nil {
		return apperror.NewNotFound("NOT_FOUND", "User not found")
	}

	if !foundUser.Password.Compare(input.CurrentPassword) {
		return apperror.NewValidation([]apperror.FieldError{
			{Field: "currentPassword", Message: "Current password is incorrect"},
		})
	}

	newPassword, err := user.NewPassword(input.NewPassword)
	if err != nil {
		return apperror.NewValidation([]apperror.FieldError{
			{Field: "newPassword", Message: err.Error()},
		})
	}

	if foundUser.Password.Compare(input.NewPassword) {
		return apperror.NewValidation([]apperror.FieldError{
			{Field: "newPassword", Message: "New password must be different from current password"},
		})
	}

	now := time.Now()

	if err := uc.sessionInvalidator.InvalidateAllUserSessions(ctx, foundUser.ID, now); err != nil {
		slog.Error("failed to invalidate sessions before password update", "user_id", foundUser.ID, "error", err)
		return apperror.NewInternal()
	}

	foundUser.UpdatePassword(newPassword)
	if err := uc.repo.Update(ctx, foundUser); err != nil {
		return apperror.NewInternal()
	}

	if err := uc.emailSender.Send(ctx, notification.EmailMessage{
		To:      foundUser.Email.String(),
		Subject: "Security Alert: Password Changed",
		Body:    "Your password has been changed successfully. If you did not make this change, please contact support immediately.",
	}); err != nil {
		slog.Error("failed to send password changed email", "user_id", foundUser.ID, "email", foundUser.Email.String(), "error", err)
	}

	return nil
}
