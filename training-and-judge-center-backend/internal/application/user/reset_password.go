package user

import (
	"context"
	"time"

	"github.com/training-judge-center/backend/internal/domain/user"
	"github.com/training-judge-center/backend/pkg/apperror"
)

var errInvalidRecoveryAttempt = apperror.NewBadRequest("INVALID_RECOVERY_ATTEMPT", "Invalid email or recovery code")

type ResetPasswordInput struct {
	Email       string
	Code        string
	NewPassword string
}

type ResetPasswordUseCase struct {
	userRepo           user.UserRepository
	recoveryRepo       user.PasswordRecoveryRepository
	sessionInvalidator user.SessionInvalidator
}

func NewResetPasswordUseCase(
	userRepo user.UserRepository,
	recoveryRepo user.PasswordRecoveryRepository,
	sessionInvalidator user.SessionInvalidator,
) *ResetPasswordUseCase {
	return &ResetPasswordUseCase{
		userRepo:           userRepo,
		recoveryRepo:       recoveryRepo,
		sessionInvalidator: sessionInvalidator,
	}
}

func (uc *ResetPasswordUseCase) Execute(ctx context.Context, input ResetPasswordInput) error {
	emailVO, err := user.NewEmail(input.Email)
	if err != nil {
		return apperror.NewValidation([]apperror.FieldError{
			{Field: "email", Message: err.Error()},
		})
	}

	foundUser, err := uc.userRepo.FindByEmail(ctx, emailVO)
	if err != nil {
		return apperror.NewInternal()
	}
	if foundUser == nil || foundUser.Status == user.StatusDeactivated {
		return errInvalidRecoveryAttempt
	}

	req, err := uc.recoveryRepo.FindPendingByUserID(ctx, foundUser.ID)
	if err != nil {
		return apperror.NewInternal()
	}
	if req == nil {
		return errInvalidRecoveryAttempt
	}

	now := time.Now()
	if req.IsExpired(now) || req.Code != input.Code {
		return errInvalidRecoveryAttempt
	}

	newPassword, err := user.NewPassword(input.NewPassword)
	if err != nil {
		return apperror.NewValidation([]apperror.FieldError{
			{Field: "newPassword", Message: err.Error()},
		})
	}

	if err := uc.sessionInvalidator.InvalidateAllUserSessions(ctx, foundUser.ID, now); err != nil {
		return apperror.NewInternal()
	}

	req.MarkAsUsed(now)
	if err := uc.recoveryRepo.Update(ctx, req); err != nil {
		return apperror.NewInternal()
	}

	foundUser.UpdatePassword(newPassword)
	if err := uc.userRepo.Update(ctx, foundUser); err != nil {
		return apperror.NewInternal()
	}

	return nil
}
