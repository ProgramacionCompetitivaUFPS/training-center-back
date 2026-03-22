package user

import (
	"context"
	"time"

	"github.com/training-judge-center/backend/internal/domain/user"
	"github.com/training-judge-center/backend/pkg/apperror"
)

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
		return &apperror.AppError{
			Code:       "NOT_FOUND",
			Message:    "No pending password recovery request found",
			StatusCode: 404,
		}
	}

	foundUser, err := uc.userRepo.FindByEmail(ctx, emailVO)
	if err != nil {
		return apperror.NewInternal()
	}
	if foundUser == nil || foundUser.Status == user.StatusDeactivated {
		return &apperror.AppError{
			Code:       "NOT_FOUND",
			Message:    "No pending password recovery request found",
			StatusCode: 404,
		}
	}

	req, err := uc.recoveryRepo.FindPendingByUserID(ctx, foundUser.ID)
	if err != nil {
		return apperror.NewInternal()
	}
	if req == nil {
		return &apperror.AppError{
			Code:       "NOT_FOUND",
			Message:    "No pending password recovery request found",
			StatusCode: 404,
		}
	}

	now := time.Now()
	if req.IsExpired(now) || req.Code != input.Code {
		return &apperror.AppError{
			Code:       "INVALID_CODE",
			Message:    "The recovery code is invalid or has expired",
			StatusCode: 400,
		}
	}

	newPassword, err := user.NewPassword(input.NewPassword)
	if err != nil {
		return apperror.NewValidation([]apperror.FieldError{
			{Field: "newPassword", Message: err.Error()},
		})
	}

	foundUser.Password = newPassword
	foundUser.UpdatedAt = &now

	if err := uc.userRepo.Update(ctx, foundUser); err != nil {
		return apperror.NewInternal()
	}

	req.MarkAsUsed(now)
	if err := uc.recoveryRepo.Update(ctx, req); err != nil {
		return apperror.NewInternal()
	}

	// Invalidate all sessions
	if err := uc.sessionInvalidator.InvalidateAllUserSessions(ctx, foundUser.ID, now); err != nil {
		return apperror.NewInternal()
	}

	return nil
}
