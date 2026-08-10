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
	refreshTokenRepo   user.RefreshTokenRepository
}

func NewResetPasswordUseCase(
	userRepo user.Repository,
	recoveryRepo user.PasswordRecoveryRepository,
	sessionInvalidator user.SessionInvalidator,
	txManager appshared.TransactionManager,
	refreshTokenRepo user.RefreshTokenRepository,
) *ResetPasswordUseCase {
	return &ResetPasswordUseCase{
		userRepo:           userRepo,
		recoveryRepo:       recoveryRepo,
		sessionInvalidator: sessionInvalidator,
		txManager:          txManager,
		refreshTokenRepo:   refreshTokenRepo,
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
		return nil, err
	}
	if foundUser == nil || foundUser.Status() == user.StatusDeactivated {
		return nil, apperror.NewBadRequest(ErrCodeInvalidRecoveryAttempt, "Invalid email or recovery code")
	}

	req, err := uc.recoveryRepo.FindPendingByUserID(ctx, foundUser.ID())
	if err != nil {
		return nil, err
	}
	if req == nil {
		return nil, apperror.NewBadRequest(ErrCodeInvalidRecoveryAttempt, "Invalid email or recovery code")
	}

	now := time.Now()
	if req.IsExpired(now) || subtle.ConstantTimeCompare([]byte(req.Code()), []byte(input.Code)) != 1 {
		return nil, apperror.NewBadRequest(ErrCodeInvalidRecoveryAttempt, "Invalid email or recovery code")
	}

	newPassword, err := user.NewPassword(input.NewPassword)
	if err != nil {
		return nil, apperror.NewValidation([]apperror.FieldError{
			{Field: "newPassword", Message: err.Error()},
		})
	}

	if err := foundUser.UpdatePassword(newPassword, now); err != nil {
		slog.ErrorContext(ctx, "failed to update password on user domain object", "user_id", foundUser.ID(), "error", err)
		return nil, apperror.NewInternal()
	}
	if err := req.MarkAsUsed(now); err != nil {
		return nil, err
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
		return nil, err
	}

	sessionsInvalidated := true
	if err := uc.sessionInvalidator.InvalidateAllUserSessions(ctx, foundUser.ID(), now); err != nil {
		sessionsInvalidated = false
	}
	if err := uc.refreshTokenRepo.RevokeAllByUserID(ctx, foundUser.ID(), now); err != nil {
		// best-effort: password already saved, don't fail the operation.
		slog.ErrorContext(ctx, "failed to revoke refresh tokens after password reset", "user_id", foundUser.ID(), "error", err)
	}

	return &ResetPasswordOutput{SessionsInvalidated: sessionsInvalidated}, nil
}
