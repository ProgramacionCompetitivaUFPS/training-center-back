package user

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/training-judge-center/backend/internal/domain/notification"
	"github.com/training-judge-center/backend/internal/domain/user"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type RequestPasswordRecoveryInput struct {
	Email string
}

type RequestPasswordRecoveryUseCase struct {
	userRepo     user.UserRepository
	recoveryRepo user.PasswordRecoveryRepository
	emailSender  notification.EmailSender
}

func NewRequestPasswordRecoveryUseCase(
	userRepo user.UserRepository,
	recoveryRepo user.PasswordRecoveryRepository,
	emailSender notification.EmailSender,
) *RequestPasswordRecoveryUseCase {
	return &RequestPasswordRecoveryUseCase{
		userRepo:     userRepo,
		recoveryRepo: recoveryRepo,
		emailSender:  emailSender,
	}
}

func (uc *RequestPasswordRecoveryUseCase) Execute(ctx context.Context, input RequestPasswordRecoveryInput) error {
	emailVO, err := user.NewEmail(input.Email)
	if err != nil {
		return apperror.NewValidation([]apperror.FieldError{
			{Field: "email", Message: "Invalid email format"},
		})
	}

	foundUser, err := uc.userRepo.FindByEmail(ctx, emailVO)
	if err != nil {
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
		return apperror.NewInternal()
	}

	code, err := generateSixDigitCode()
	if err != nil {
		return apperror.NewInternal()
	}

	req := user.RestorePasswordRecoveryRequest(uuid.NewString(), foundUser.ID(), code, user.StatusPending, now.Add(15 * time.Minute), now, nil)

	if err := uc.recoveryRepo.Save(ctx, req); err != nil {
		return apperror.NewInternal()
	}

	if err := uc.emailSender.Send(ctx, notification.EmailMessage{
		To:      foundUser.Email().String(),
		Subject: "Password Recovery Code",
		Body:    fmt.Sprintf("Your password recovery code is: %s\nThis code will expire in 15 minutes.", code),
	}); err != nil {
		slog.Error("failed to send password recovery email", "user_id", foundUser.ID(), "error", err)
		if invalidateErr := uc.recoveryRepo.InvalidatePendingByUserID(ctx, foundUser.ID(), time.Now()); invalidateErr != nil {
			slog.Error("failed to invalidate undelivered recovery code", "user_id", foundUser.ID(), "error", invalidateErr)
		}
	}

	return nil
}
