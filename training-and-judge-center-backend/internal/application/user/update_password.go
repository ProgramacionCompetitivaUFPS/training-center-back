package user

import (
	"context"
	"log/slog"
	"time"

	appshared "github.com/training-judge-center/backend/internal/application/shared"
	"github.com/training-judge-center/backend/internal/domain/user"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type UpdatePasswordInput struct {
	UserID          string
	CurrentPassword string
	NewPassword     string
}

type UpdatePasswordOutput struct {
	SessionsInvalidated bool
}

type UpdatePasswordUseCase struct {
	repo               user.Repository
	emailSender        appshared.EmailSender
	sessionInvalidator user.SessionInvalidator
	rateLimiter        appshared.RateLimiter
}

func NewUpdatePasswordUseCase(repo user.Repository, email appshared.EmailSender, sessionInvalidator user.SessionInvalidator, rateLimiter appshared.RateLimiter) *UpdatePasswordUseCase {
	return &UpdatePasswordUseCase{
		repo:               repo,
		emailSender:        email,
		sessionInvalidator: sessionInvalidator,
		rateLimiter:        rateLimiter,
	}
}

func (uc *UpdatePasswordUseCase) Execute(ctx context.Context, input UpdatePasswordInput) (*UpdatePasswordOutput, error) {
	rateKey := "rate_limit:update_password:" + input.UserID
	allowed, err := uc.rateLimiter.Allow(ctx, rateKey, 5, time.Hour)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, apperror.NewTooManyRequests(ErrCodeTooManyRequests, "Too many password update attempts. Please try again in an hour.", 3600)
	}

	foundUser, err := uc.repo.FindByID(ctx, input.UserID)
	if err != nil {
		return nil, err
	}
	if foundUser == nil {
		return nil, apperror.NewNotFound(user.ErrCodeUserNotFound, "User not found")
	}

	if !foundUser.Password().Compare(input.CurrentPassword) {
		return nil, apperror.NewValidation([]apperror.FieldError{
			{Field: "currentPassword", Message: "Current password is incorrect"},
		})
	}

	newPassword, err := user.NewPassword(input.NewPassword)
	if err != nil {
		return nil, apperror.NewValidation([]apperror.FieldError{
			{Field: "newPassword", Message: err.Error()},
		})
	}

	if foundUser.Password().Compare(input.NewPassword) {
		return nil, apperror.NewValidation([]apperror.FieldError{
			{Field: "newPassword", Message: "New password must be different from current password"},
		})
	}

	now := time.Now()
	if err := foundUser.UpdatePassword(newPassword, now); err != nil {
		slog.ErrorContext(ctx, "failed to update password on user domain object", "user_id", foundUser.ID(), "error", err)
		return nil, apperror.NewInternal()
	}

	if err := uc.repo.Update(ctx, foundUser); err != nil {
		return nil, err
	}

	// Password changed successfully — reset rate limit so the user can make fresh attempts later.
	if err := uc.rateLimiter.Reset(ctx, rateKey); err != nil {
		// Don't fail the operation; the password was already saved.
		_ = err
	}

	sessionsInvalidated := true
	if err := uc.sessionInvalidator.InvalidateAllUserSessions(ctx, foundUser.ID(), now); err != nil {
		sessionsInvalidated = false
	}

	_ = uc.emailSender.Send(ctx, appshared.EmailMessage{
		To:      foundUser.Email().String(),
		Subject: "Security Alert: Password Changed",
		Body:    "Your password has been changed successfully. If you did not make this change, please contact support immediately.",
	})

	return &UpdatePasswordOutput{SessionsInvalidated: sessionsInvalidated}, nil
}
