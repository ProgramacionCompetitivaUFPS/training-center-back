package user

import (
	"context"
	"crypto/subtle"
	"log/slog"
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
	txManager          TransactionManager
}

func NewResetPasswordUseCase(
	userRepo user.UserRepository,
	recoveryRepo user.PasswordRecoveryRepository,
	sessionInvalidator user.SessionInvalidator,
	txManager TransactionManager,
) *ResetPasswordUseCase {
	return &ResetPasswordUseCase{
		userRepo:           userRepo,
		recoveryRepo:       recoveryRepo,
		sessionInvalidator: sessionInvalidator,
		txManager:          txManager,
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
		slog.Error("failed to find user by email during password reset", "error", err)
		return apperror.NewInternal()
	}
	if foundUser == nil || foundUser.Status() == user.StatusDeactivated {
		return errInvalidRecoveryAttempt
	}

	req, err := uc.recoveryRepo.FindPendingByUserID(ctx, foundUser.ID())
	if err != nil {
		slog.Error("failed to find pending recovery request", "user_id", foundUser.ID(), "error", err)
		return apperror.NewInternal()
	}
	if req == nil {
		return errInvalidRecoveryAttempt
	}

	now := time.Now()
	if req.IsExpired(now) || subtle.ConstantTimeCompare([]byte(req.Code()), []byte(input.Code)) != 1 {
		return errInvalidRecoveryAttempt
	}

	newPassword, err := user.NewPassword(input.NewPassword)
	if err != nil {
		return apperror.NewValidation([]apperror.FieldError{
			{Field: "newPassword", Message: err.Error()},
		})
	}

	if err := foundUser.UpdatePassword(newPassword); err != nil {
		slog.Error("failed to update password on user domain object", "user_id", foundUser.ID(), "error", err)
		return apperror.NewInternal()
	}
	if err := req.MarkAsUsed(now); err != nil {
		slog.Error("failed to mark recovery request as used", "user_id", foundUser.ID(), "error", err)
		return apperror.NewInternal()
	}

	if err := uc.txManager.WithTx(ctx, func(txCtx context.Context) error {
		if err := uc.userRepo.Update(txCtx, foundUser); err != nil {
			return err
		}
		if err := uc.recoveryRepo.Update(txCtx, req); err != nil {
			return err
		}
		return nil
	}); err != nil {
		slog.Error("failed to commit password reset transaction", "user_id", foundUser.ID(), "error", err)
		return apperror.NewInternal()
	}

	if err := uc.sessionInvalidator.InvalidateAllUserSessions(ctx, foundUser.ID(), now); err != nil {
		slog.Error("failed to invalidate sessions after password reset", "user_id", foundUser.ID(), "error", err)
		return ErrSessionsNotInvalidated
	}

	return nil
}
