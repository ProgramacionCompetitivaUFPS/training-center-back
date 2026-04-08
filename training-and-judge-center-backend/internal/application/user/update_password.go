package user

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/training-judge-center/backend/internal/domain/notification"
	"github.com/training-judge-center/backend/internal/domain/ratelimit"
	"github.com/training-judge-center/backend/internal/domain/user"
	"github.com/training-judge-center/backend/pkg/apperror"
)

// ErrSessionsNotInvalidated is returned when the password was changed successfully
// but the active sessions could not be revoked. The caller should inform the user
// that changing their password again will close the remaining sessions.
var ErrSessionsNotInvalidated = errors.New("password changed but sessions could not be invalidated")

type UpdatePasswordInput struct {
	UserID          string
	CurrentPassword string
	NewPassword     string
}

type UpdatePasswordUseCase struct {
	repo               user.UserRepository
	emailSender        notification.EmailSender
	sessionInvalidator user.SessionInvalidator
	rateLimiter        ratelimit.RateLimiter
}

func NewUpdatePasswordUseCase(repo user.UserRepository, email notification.EmailSender, sessionInvalidator user.SessionInvalidator, rateLimiter ratelimit.RateLimiter) *UpdatePasswordUseCase {
	return &UpdatePasswordUseCase{
		repo:               repo,
		emailSender:        email,
		sessionInvalidator: sessionInvalidator,
		rateLimiter:        rateLimiter,
	}
}

func (uc *UpdatePasswordUseCase) Execute(ctx context.Context, input UpdatePasswordInput) error {
	rateKey := "rate_limit:update_password:" + input.UserID
	allowed, err := uc.rateLimiter.Allow(ctx, rateKey, 5, time.Hour)
	if err != nil {
		slog.Error("failed to check rate limit for password update", "user_id", input.UserID, "error", err)
		return apperror.NewInternal()
	}
	if !allowed {
		return apperror.NewTooManyRequests("TOO_MANY_REQUESTS", "Too many password update attempts. Please try again in an hour.", 3600)
	}

	foundUser, err := uc.repo.FindByID(ctx, input.UserID)
	if err != nil {
		slog.Error("failed to find user during password update", "user_id", input.UserID, "error", err)
		return apperror.NewInternal()
	}
	if foundUser == nil {
		return apperror.NewNotFound("NOT_FOUND", "User not found")
	}

	if !foundUser.Password().Compare(input.CurrentPassword) {
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

	if foundUser.Password().Compare(input.NewPassword) {
		return apperror.NewValidation([]apperror.FieldError{
			{Field: "newPassword", Message: "New password must be different from current password"},
		})
	}

	if err := foundUser.UpdatePassword(newPassword); err != nil {
		slog.Error("failed to update password on user domain object", "user_id", foundUser.ID(), "error", err)
		return apperror.NewInternal()
	}

	if err := uc.repo.Update(ctx, foundUser); err != nil {
		slog.Error("failed to persist password update", "user_id", foundUser.ID(), "error", err)
		return apperror.NewInternal()
	}

	// Password changed successfully — reset rate limit so the user can make fresh attempts later.
	if err := uc.rateLimiter.Reset(ctx, rateKey); err != nil {
		slog.Error("failed to reset rate limit after successful password update", "user_id", foundUser.ID(), "error", err)
		// Don't fail the operation; the password was already saved.
	}

	now := time.Now()
	if err := uc.sessionInvalidator.InvalidateAllUserSessions(ctx, foundUser.ID(), now); err != nil {
		slog.Error("failed to invalidate sessions after password update", "user_id", foundUser.ID(), "error", err)
		return ErrSessionsNotInvalidated
	}

	if err := uc.emailSender.Send(ctx, notification.EmailMessage{
		To:      foundUser.Email().String(),
		Subject: "Security Alert: Password Changed",
		Body:    "Your password has been changed successfully. If you did not make this change, please contact support immediately.",
	}); err != nil {
		slog.Error("failed to send password changed email", "user_id", foundUser.ID(), "email", foundUser.Email().String(), "error", err)
	}

	return nil
}
