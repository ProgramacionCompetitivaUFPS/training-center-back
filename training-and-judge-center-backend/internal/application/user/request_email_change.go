package user

import (
	"context"
	"crypto/rand"
	"fmt"
	"log/slog"
	"math/big"
	"time"

	"github.com/google/uuid"
	"github.com/training-judge-center/backend/internal/domain/notification"
	"github.com/training-judge-center/backend/internal/domain/user"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type RequestEmailChangeInput struct {
	UserID   string
	Password string
	NewEmail string
}

type RequestEmailChangeUseCase struct {
	userRepo        user.UserRepository
	emailChangeRepo user.EmailChangeRepository
	emailSender     notification.EmailSender
}

func NewRequestEmailChangeUseCase(
	userRepo user.UserRepository,
	emailChangeRepo user.EmailChangeRepository,
	emailSender notification.EmailSender,
) *RequestEmailChangeUseCase {
	return &RequestEmailChangeUseCase{
		userRepo:        userRepo,
		emailChangeRepo: emailChangeRepo,
		emailSender:     emailSender,
	}
}

func (uc *RequestEmailChangeUseCase) Execute(ctx context.Context, input RequestEmailChangeInput) error {
	u, err := uc.userRepo.FindByID(ctx, input.UserID)
	if err != nil {
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

	emailExists, err := uc.userRepo.ExistsByEmail(ctx, parsedNewEmail)
	if err != nil {
		return apperror.NewInternal()
	}
	if emailExists {
		return apperror.NewConflict("EMAIL_ALREADY_EXISTS", "The email address is already in use")
	}

	if err := uc.emailChangeRepo.InvalidatePendingByUserID(ctx, input.UserID); err != nil {
		return apperror.NewInternal()
	}

	code, err := generateSixDigitCode()
	if err != nil {
		return apperror.NewInternal()
	}

	now := time.Now()
	expiresAt := now.Add(15 * time.Minute)

	req := user.RestoreEmailChangeRequest(uuid.New().String(), input.UserID, parsedNewEmail, code, user.StatusPending, expiresAt, now, nil)

	if err := uc.emailChangeRepo.Save(ctx, req); err != nil {
		return apperror.NewInternal()
	}

	if err := uc.emailSender.Send(ctx, notification.EmailMessage{
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
