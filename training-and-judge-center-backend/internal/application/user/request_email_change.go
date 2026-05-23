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

type RequestEmailChangeInput struct {
	UserID   string
	Password string
	NewEmail string
}

type RequestEmailChangeOutput struct {
	ExpiresAt time.Time
}

type RequestEmailChangeUseCase struct {
	userRepo        user.Repository
	emailChangeRepo user.EmailChangeRepository
	emailSender     appshared.EmailSender
	rateLimiter     appshared.RateLimiter
}

func NewRequestEmailChangeUseCase(
	userRepo user.Repository,
	emailChangeRepo user.EmailChangeRepository,
	emailSender appshared.EmailSender,
	rateLimiter appshared.RateLimiter,
) *RequestEmailChangeUseCase {
	return &RequestEmailChangeUseCase{
		userRepo:        userRepo,
		emailChangeRepo: emailChangeRepo,
		emailSender:     emailSender,
		rateLimiter:     rateLimiter,
	}
}

func (uc *RequestEmailChangeUseCase) Execute(ctx context.Context, input RequestEmailChangeInput) (*RequestEmailChangeOutput, error) {
	allowed, err := uc.rateLimiter.Allow(ctx, "email-change-request:"+input.UserID, 5, time.Hour)
	if err != nil {
		slog.ErrorContext(ctx, "failed to check rate limit for email change request", "user_id", input.UserID, "error", err)
		return nil, apperror.NewInternal()
	}
	if !allowed {
		return nil, apperror.NewTooManyRequests(ErrCodeTooManyRequests, "Too many requests. Please try again later.", 3600)
	}

	u, err := uc.userRepo.FindByID(ctx, input.UserID)
	if err != nil {
		slog.ErrorContext(ctx, "failed to find user during email change request", "user_id", input.UserID, "error", err)
		return nil, apperror.NewInternal()
	}
	if u == nil {
		return nil, apperror.NewUnauthorized(ErrCodeInvalidCredentials, "Invalid credentials")
	}

	if !u.Password().Compare(input.Password) {
		return nil, apperror.NewUnauthorized(ErrCodeInvalidCredentials, "Invalid credentials")
	}

	parsedNewEmail, err := user.NewEmail(input.NewEmail)
	if err != nil {
		return nil, apperror.NewValidation([]apperror.FieldError{
			{Field: "newEmail", Message: err.Error()},
		})
	}

	now := time.Now()

	if err := uc.emailChangeRepo.InvalidatePendingByUserID(ctx, input.UserID, now); err != nil {
		slog.ErrorContext(ctx, "failed to invalidate pending email change requests", "user_id", input.UserID, "error", err)
		return nil, apperror.NewInternal()
	}

	code, err := generateSixDigitCode()
	if err != nil {
		slog.ErrorContext(ctx, "failed to generate email change code", "user_id", input.UserID, "error", err)
		return nil, apperror.NewInternal()
	}

	newID := uuid.New().String()
	req, err := user.NewEmailChangeRequest(newID, input.UserID, parsedNewEmail, code, now)
	if err != nil {
		slog.ErrorContext(ctx, "failed to build email change request", "user_id", input.UserID, "error", err)
		return nil, apperror.NewInternal()
	}

	if err := uc.emailChangeRepo.Save(ctx, req); err != nil {
		slog.ErrorContext(ctx, "failed to save email change request", "user_id", input.UserID, "error", err)
		return nil, apperror.NewInternal()
	}

	if err := uc.emailSender.Send(ctx, appshared.EmailMessage{
		To:      input.NewEmail,
		Subject: "Verify your new email address",
		Body:    fmt.Sprintf("Your email verification code is: %s. It will expire in 15 minutes.", code),
	}); err != nil {
		slog.ErrorContext(ctx, "failed to send verification email", "email", input.NewEmail, "error", err)
		return nil, apperror.NewServiceUnavailable(ErrCodeEmailDeliveryFailed, "We couldn't deliver the verification code to your email. Please try again later.")
	}

	return &RequestEmailChangeOutput{ExpiresAt: req.ExpiresAt()}, nil
}
