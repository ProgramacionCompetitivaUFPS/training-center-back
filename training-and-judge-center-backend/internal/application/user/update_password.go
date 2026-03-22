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

func (uc *UpdatePasswordUseCase) Execute(ctx context.Context, userID string, input UpdatePasswordInput) error {
	foundUser, err := uc.repo.FindByID(ctx, userID)
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
	foundUser.Password = newPassword
	foundUser.UpdatedAt = &now

	if err := uc.repo.Update(ctx, foundUser); err != nil {
		return apperror.NewInternal()
	}

	// Invalidate all active sessions for this user
	if err := uc.sessionInvalidator.InvalidateAllUserSessions(ctx, foundUser.ID, now); err != nil {
		slog.Error("failed to invalidate sessions after password update", "user_id", foundUser.ID, "error", err)
	}

	subject := "Security Alert: Password Changed"
	body := "Your password has been changed successfully. If you did not make this change, please contact support immediately."
	_ = uc.emailSender.Send(ctx, foundUser.Email.String(), subject, body)

	return nil
}
