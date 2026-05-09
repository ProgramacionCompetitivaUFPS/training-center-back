package user

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	appshared "github.com/training-judge-center/backend/internal/application/shared"
	"github.com/training-judge-center/backend/internal/domain/user"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type RequestPasswordRecoveryInput struct {
	Email string
}

type RequestPasswordRecoveryUseCase struct {
	userRepo     user.Repository
	recoveryRepo user.PasswordRecoveryRepository
	emailSender  appshared.EmailSender
	rateLimiter  appshared.RateLimiter
}

func NewRequestPasswordRecoveryUseCase(
	userRepo user.Repository,
	recoveryRepo user.PasswordRecoveryRepository,
	emailSender appshared.EmailSender,
	rateLimiter appshared.RateLimiter,
) *RequestPasswordRecoveryUseCase {
	return &RequestPasswordRecoveryUseCase{
		userRepo:     userRepo,
		recoveryRepo: recoveryRepo,
		emailSender:  emailSender,
		rateLimiter:  rateLimiter,
	}
}

func (uc *RequestPasswordRecoveryUseCase) Execute(ctx context.Context, input RequestPasswordRecoveryInput) error {
	allowed, err := uc.rateLimiter.Allow(ctx, "password-recovery:"+input.Email, 5, time.Hour)
	if err != nil {
		slog.Error("failed to check rate limit for password recovery request", "email", input.Email, "error", err)
		return apperror.NewInternal()
	}
	if !allowed {
		return apperror.NewTooManyRequests("RATE_LIMIT_EXCEEDED", "Too many recovery requests. Please try again later.", 3600)
	}

	emailVO, err := user.NewEmail(input.Email)
	if err != nil {
		return apperror.NewValidation([]apperror.FieldError{
			{Field: "email", Message: "Invalid email format"},
		})
	}

	foundUser, err := uc.userRepo.FindByEmail(ctx, emailVO)
	if err != nil {
		slog.Error("failed to find user by email during password recovery request", "error", err)
		return apperror.NewInternal()
	}

	// Ambiguous response: If user doesn't exist, we just return nil natively 
	// The HTTP handler will always return 200 OK.
	if foundUser == nil || foundUser.Status() == user.StatusDeactivated {
		return nil
	}

	now := time.Now()

	// Invalidate previous requests
	if err := uc.recoveryRepo.InvalidatePendingByUserID(ctx, foundUser.ID(), now); err != nil {
		slog.Error("failed to invalidate pending recovery requests", "user_id", foundUser.ID(), "error", err)
		return apperror.NewInternal()
	}

	code, err := generateSixDigitCode()
	if err != nil {
		slog.Error("failed to generate recovery code", "user_id", foundUser.ID(), "error", err)
		return apperror.NewInternal()
	}

	newID := uuid.New().String()
	req, err := user.NewPasswordRecoveryRequest(newID, foundUser.ID(), code, now)
	if err != nil {
		slog.Error("failed to build password recovery request", "user_id", foundUser.ID(), "error", err)
		return apperror.NewInternal()
	}

	if err := uc.recoveryRepo.Save(ctx, req); err != nil {
		slog.Error("failed to save password recovery request", "user_id", foundUser.ID(), "error", err)
		return apperror.NewInternal()
	}

	if err := uc.emailSender.Send(ctx, appshared.EmailMessage{
		To:      foundUser.Email().String(),
		Subject: "Password Recovery Code",
		Body:    fmt.Sprintf("Your password recovery code is: %s\nThis code will expire in 15 minutes.", code),
	}); err != nil {
		slog.Error("failed to send password recovery email", "user_id", foundUser.ID(), "error", err)
		if invalidateErr := uc.recoveryRepo.InvalidatePendingByUserID(ctx, foundUser.ID(), now); invalidateErr != nil {
			slog.Error("failed to invalidate undelivered recovery code", "user_id", foundUser.ID(), "error", invalidateErr)
		}
	}

	return nil
}
