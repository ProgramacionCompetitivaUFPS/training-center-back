package user

import (
	"context"
	"crypto/subtle"
	"log/slog"
	"time"

	appshared "github.com/training-judge-center/backend/internal/application/shared"
	"github.com/training-judge-center/backend/internal/domain/user"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type ResetPasswordInput struct {
	Email       string
	Code        string
	NewPassword string
}

type ResetPasswordOutput struct {
	SessionsInvalidated bool
}

type ResetPasswordUseCase struct {
	userRepo           user.Repository
	recoveryRepo       user.PasswordRecoveryRepository
	sessionInvalidator user.SessionInvalidator
	txManager          appshared.TransactionManager
}

func NewResetPasswordUseCase(
	userRepo user.Repository,
	recoveryRepo user.PasswordRecoveryRepository,
	sessionInvalidator user.SessionInvalidator,
	txManager appshared.TransactionManager,
) *ResetPasswordUseCase {
	return &ResetPasswordUseCase{
		userRepo:           userRepo,
		recoveryRepo:       recoveryRepo,
		sessionInvalidator: sessionInvalidator,
		txManager:          txManager,
	}
}

func (uc *ResetPasswordUseCase) Execute(ctx context.Context, input ResetPasswordInput) (*ResetPasswordOutput, error) {
	emailVO, err := user.NewEmail(input.Email)
	if err != nil {
		return nil, apperror.NewValidation([]apperror.FieldError{
			{Field: "email", Message: err.Error()},
		})
	}

	foundUser, err := uc.userRepo.FindByEmail(ctx, emailVO)
	if err != nil {
		slog.Error("failed to find user by email during password reset", "error", err)
		return nil, apperror.NewInternal()
	}
	if foundUser == nil || foundUser.Status() == user.StatusDeactivated {
		return nil, errInvalidRecoveryAttempt
	}

	req, err := uc.recoveryRepo.FindPendingByUserID(ctx, foundUser.ID())
	if err != nil {
		slog.Error("failed to find pending recovery request", "user_id", foundUser.ID(), "error", err)
		return nil, apperror.NewInternal()
	}
	if req == nil {
		return nil, errInvalidRecoveryAttempt
	}

	now := time.Now()
	if req.IsExpired(now) || subtle.ConstantTimeCompare([]byte(req.Code()), []byte(input.Code)) != 1 {
		return nil, errInvalidRecoveryAttempt
	}

	newPassword, err := user.NewPassword(input.NewPassword)
	if err != nil {
		return nil, apperror.NewValidation([]apperror.FieldError{
			{Field: "newPassword", Message: err.Error()},
		})
	}

	if err := foundUser.UpdatePassword(newPassword, now); err != nil {
		slog.Error("failed to update password on user domain object", "user_id", foundUser.ID(), "error", err)
		return nil, apperror.NewInternal()
	}
	if err := req.MarkAsUsed(now); err != nil {
		slog.Error("failed to mark recovery request as used", "user_id", foundUser.ID(), "error", err)
		return nil, apperror.NewInternal()
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
		return nil, apperror.NewInternal()
	}

	sessionsInvalidated := true
	if err := uc.sessionInvalidator.InvalidateAllUserSessions(ctx, foundUser.ID(), now); err != nil {
		slog.Error("failed to invalidate sessions after password reset", "user_id", foundUser.ID(), "error", err)
		sessionsInvalidated = false
	}

	return &ResetPasswordOutput{SessionsInvalidated: sessionsInvalidated}, nil
}
