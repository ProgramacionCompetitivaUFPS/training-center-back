package user

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	appshared "github.com/training-judge-center/backend/internal/application/shared"
	domainShared "github.com/training-judge-center/backend/internal/domain/shared"
	"github.com/training-judge-center/backend/internal/domain/user"
	"github.com/training-judge-center/backend/pkg/apperror"
	"github.com/training-judge-center/backend/pkg/emailtemplate"
)

type RequestDeactivationInput struct {
	UserID string
	// Current password. Required when the account has one; Google-only accounts (no password)
	// confirm with the emailed code alone.
	Password string
}

// Same budget as UpdatePassword: checking the password here must not become a way to guess it.
const (
	requestDeactivationRateLimitMaxAttempts = 5
	requestDeactivationRateLimitWindow      = time.Hour
)

type RequestDeactivationUseCase struct {
	userRepo    user.Repository
	deactRepo   user.DeactivationRequestRepository
	emailSender appshared.EmailSender
	rateLimiter appshared.RateLimiter
}

func NewRequestDeactivationUseCase(
	userRepo user.Repository,
	deactRepo user.DeactivationRequestRepository,
	emailSender appshared.EmailSender,
	rateLimiter appshared.RateLimiter,
) *RequestDeactivationUseCase {
	return &RequestDeactivationUseCase{
		userRepo:    userRepo,
		deactRepo:   deactRepo,
		emailSender: emailSender,
		rateLimiter: rateLimiter,
	}
}

func (uc *RequestDeactivationUseCase) Execute(ctx context.Context, input RequestDeactivationInput) error {
	foundUser, err := uc.userRepo.FindByID(ctx, input.UserID)
	if err != nil {
		return err
	}
	if foundUser == nil || foundUser.Status() == user.StatusDeactivated {
		return apperror.NewNotFound(user.ErrCodeUserNotFound, "User not found")
	}

	if foundUser.Role() == domainShared.RoleAdmin {
		return apperror.NewForbidden(ErrCodeAdminCannotRequestDeactivation, "Administrators cannot deactivate their own account")
	}

	// The form asks for the current password and the spec (FL-USR-06) says it's checked; it
	// wasn't, so any text went through. Accounts without a password rely on the emailed code.
	if foundUser.Password().HasPassword() {
		allowed, err := uc.rateLimiter.Allow(ctx, "rate_limit:request_deactivation:"+foundUser.ID(),
			requestDeactivationRateLimitMaxAttempts, requestDeactivationRateLimitWindow)
		if err != nil {
			return err
		}
		if !allowed {
			return apperror.NewTooManyRequests(ErrCodeTooManyRequests,
				"Too many deactivation attempts. Please try again in an hour.", int(requestDeactivationRateLimitWindow.Seconds()))
		}
		if !foundUser.Password().Compare(input.Password) {
			return apperror.NewValidation([]apperror.FieldError{
				{Field: "password", Message: "Current password is incorrect"},
			})
		}
	}

	now := time.Now()

	// Invalidate previous requests to ensure only one code is active
	if err := uc.deactRepo.InvalidatePendingByUserID(ctx, foundUser.ID(), now); err != nil {
		return err
	}

	code, err := generateSixDigitCode()
	if err != nil {
		slog.ErrorContext(ctx, "failed to generate deactivation code", "user_id", foundUser.ID(), "error", err)
		return apperror.NewInternal()
	}

	newID := uuid.New().String()
	req, err := user.NewDeactivationRequest(newID, foundUser.ID(), code, now)
	if err != nil {
		slog.ErrorContext(ctx, "failed to build deactivation request", "user_id", foundUser.ID(), "error", err)
		return apperror.NewInternal()
	}

	if err := uc.deactRepo.Save(ctx, req); err != nil {
		return err
	}

	htmlContent := "<p style=\"margin:0 0 8px;\">We received a request to deactivate your Training Center account.</p>" +
		"<p style=\"margin:0 0 4px;\">Enter the confirmation code below to proceed:</p>" +
		emailtemplate.CodeBlock(code) +
		"<p style=\"margin:0 0 12px;color:#64748b;font-size:14px;\">This code expires in <strong>15 minutes</strong>.</p>" +
		"<p style=\"margin:0;color:#b91c1c;font-size:14px;\"><strong>Warning:</strong> Confirming this code will permanently anonymize your account and log you out of all sessions.</p>"
	if err := uc.emailSender.Send(ctx, appshared.EmailMessage{
		To:       foundUser.Email().String(),
		Subject:  "Account Deactivation Code",
		Body:     fmt.Sprintf("You requested to deactivate your account.\n\nYour confirmation code is: %s\n\nThis code will expire in 15 minutes. Note: Confirming this code will completely anonymize your account and log you out immediately.", code),
		HTMLBody: emailtemplate.Wrap("Account Deactivation Code", htmlContent),
	}); err != nil {
		return apperror.NewServiceUnavailable(ErrCodeEmailDeliveryFailed, "Failed to send verification email")
	}

	return nil
}
