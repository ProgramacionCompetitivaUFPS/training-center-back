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
		return err
	}
	if !allowed {
		return apperror.NewTooManyRequests(ErrCodeTooManyRequests, "Too many recovery requests. Please try again later.", 3600)
	}

	emailVO, err := user.NewEmail(input.Email)
	if err != nil {
		return apperror.NewValidation([]apperror.FieldError{
			{Field: "email", Message: "Invalid email format"},
		})
	}

	foundUser, err := uc.userRepo.FindByEmail(ctx, emailVO)
	if err != nil {
		return err
	}

	// Ambiguous response: If user doesn't exist, we just return nil natively 
	// The HTTP handler will always return 200 OK.
	if foundUser == nil || foundUser.Status() == user.StatusDeactivated {
		return nil
	}

	now := time.Now()

	// Invalidate previous requests
	if err := uc.recoveryRepo.InvalidatePendingByUserID(ctx, foundUser.ID(), now); err != nil {
		return err
	}

	code, err := generateSixDigitCode()
	if err != nil {
		slog.ErrorContext(ctx, "failed to generate recovery code", "user_id", foundUser.ID(), "error", err)
		return apperror.NewInternal()
	}

	newID := uuid.New().String()
	req, err := user.NewPasswordRecoveryRequest(newID, foundUser.ID(), code, now)
	if err != nil {
		slog.ErrorContext(ctx, "failed to build password recovery request", "user_id", foundUser.ID(), "error", err)
		return apperror.NewInternal()
	}

	if err := uc.recoveryRepo.Save(ctx, req); err != nil {
		return err
	}

	if err := uc.emailSender.Send(ctx, appshared.EmailMessage{
		To:      foundUser.Email().String(),
		Subject: "Password Recovery Code",
		Body:    fmt.Sprintf("Your password recovery code is: %s\nThis code will expire in 15 minutes.", code),
	}); err != nil {
		_ = uc.recoveryRepo.InvalidatePendingByUserID(ctx, foundUser.ID(), now)
	}

	return nil
}
