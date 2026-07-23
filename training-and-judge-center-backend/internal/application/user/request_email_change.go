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
	"github.com/training-judge-center/backend/pkg/emailtemplate"
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
		return nil, err
	}
	if !allowed {
		return nil, apperror.NewTooManyRequests(ErrCodeTooManyRequests, "Too many requests. Please try again later.", 3600)
	}

	u, err := uc.userRepo.FindByID(ctx, input.UserID)
	if err != nil {
		return nil, err
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
		return nil, err
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
		return nil, err
	}

	htmlContent := "<p style=\"margin:0 0 8px;\">We received a request to update the email address on your Training Center account.</p>" +
		"<p style=\"margin:0 0 4px;\">Enter the code below to verify your new email:</p>" +
		emailtemplate.CodeBlock(code) +
		"<p style=\"margin:0;color:#64748b;font-size:14px;\">This code expires in <strong>15 minutes</strong>. If you didn't request this change, please contact support immediately.</p>"
	if err := uc.emailSender.Send(ctx, appshared.EmailMessage{
		To:       input.NewEmail,
		Subject:  "Verify your new email address",
		Body:     fmt.Sprintf("Your email verification code is: %s. It will expire in 15 minutes.", code),
		HTMLBody: emailtemplate.Wrap("Verify your new email address", htmlContent),
	}); err != nil {
		return nil, apperror.NewServiceUnavailable(ErrCodeEmailDeliveryFailed, "We couldn't deliver the verification code to your email. Please try again later.")
	}

	return &RequestEmailChangeOutput{ExpiresAt: req.ExpiresAt()}, nil
}
