package user

import (
	"context"
	"crypto/rand"
	"fmt"
	"log/slog"
	"math/big"
	"time"

	"github.com/google/uuid"
	"github.com/training-judge-center/backend/internal/application/shared"
	"github.com/training-judge-center/backend/internal/domain/user"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type RequestEmailChangeInput struct {
	UserID   string
	Password string
	NewEmail string
}

type RequestEmailChangeUseCase struct {
	userRepo        user.Repository
	emailChangeRepo user.EmailChangeRepository
	emailSender     shared.EmailSender
	rateLimiter     shared.RateLimiter
}

func NewRequestEmailChangeUseCase(
	userRepo user.Repository,
	emailChangeRepo user.EmailChangeRepository,
	emailSender shared.EmailSender,
	rateLimiter shared.RateLimiter,
) *RequestEmailChangeUseCase {
	return &RequestEmailChangeUseCase{
		userRepo:        userRepo,
		emailChangeRepo: emailChangeRepo,
		emailSender:     emailSender,
		rateLimiter:     rateLimiter,
	}
}

func (uc *RequestEmailChangeUseCase) Execute(ctx context.Context, input RequestEmailChangeInput) error {
	allowed, err := uc.rateLimiter.Allow(ctx, "email-change-request:"+input.UserID, 5, time.Hour)
	if err != nil {
		slog.Error("failed to check rate limit for email change request", "user_id", input.UserID, "error", err)
		return apperror.NewInternal()
	}
	if !allowed {
		return apperror.NewTooManyRequests("RATE_LIMIT_EXCEEDED", "Too many requests. Please try again later.", 3600)
	}

	u, err := uc.userRepo.FindByID(ctx, input.UserID)
	if err != nil {
		slog.Error("failed to find user during email change request", "user_id", input.UserID, "error", err)
		return apperror.NewInternal()
	}
	if u == nil {
		return apperror.NewUnauthorized("INVALID_CREDENTIALS", "Invalid credentials")
	}

	if !u.Password().Compare(input.Password) {
		return apperror.NewUnauthorized("INVALID_CREDENTIALS", "Invalid credentials")
	}

	parsedNewEmail, err := user.NewEmail(input.NewEmail)
	if err != nil {
		return apperror.NewValidation([]apperror.FieldError{
			{Field: "newEmail", Message: err.Error()},
		})
	}

	if err := uc.emailChangeRepo.InvalidatePendingByUserID(ctx, input.UserID); err != nil {
		slog.Error("failed to invalidate pending email change requests", "user_id", input.UserID, "error", err)
		return apperror.NewInternal()
	}

	code, err := generateSixDigitCode()
	if err != nil {
		slog.Error("failed to generate email change code", "user_id", input.UserID, "error", err)
		return apperror.NewInternal()
	}

	now := time.Now()

	req, err := user.NewEmailChangeRequest(uuid.New().String(), input.UserID, parsedNewEmail, code, now)
	if err != nil {
		slog.Error("failed to build email change request", "user_id", input.UserID, "error", err)
		return apperror.NewInternal()
	}

	if err := uc.emailChangeRepo.Save(ctx, req); err != nil {
		slog.Error("failed to save email change request", "user_id", input.UserID, "error", err)
		return apperror.NewInternal()
	}

	if err := uc.emailSender.Send(ctx, shared.EmailMessage{
		To:      input.NewEmail,
		Subject: "Verify your new email address",
		Body:    fmt.Sprintf("Your email verification code is: %s. It will expire in 15 minutes.", code),
	}); err != nil {
		slog.Error("failed to send verification email", "email", input.NewEmail, "error", err)
		return apperror.NewServiceUnavailable("EMAIL_DELIVERY_FAILED", "We couldn't deliver the verification code to your email. Please try again later.")
	}

	return nil
}

func generateSixDigitCode() (string, error) {
	// Generate a random number between 0 and 999999
	max := big.NewInt(1000000)
	n, err := rand.Int(rand.Reader, max)
	if err != nil {
		return "", err
	}
	// Format as a 6-digit string with leading zeros
	return fmt.Sprintf("%06d", n.Int64()), nil
}
