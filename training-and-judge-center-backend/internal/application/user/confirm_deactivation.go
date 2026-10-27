package user

import (
	"context"
	"crypto/subtle"
	"log/slog"
	"time"

	"github.com/google/uuid"
	appshared "github.com/training-judge-center/backend/internal/application/shared"
	"github.com/training-judge-center/backend/internal/domain/user"
	"github.com/training-judge-center/backend/pkg/apperror"
	"github.com/training-judge-center/backend/pkg/emailtemplate"
)

type ConfirmDeactivationInput struct {
	UserID    string
	Code      string
	IP        *string
	UserAgent *string
}

type ConfirmDeactivationOutput struct {
	SessionsInvalidated bool
}

type ConfirmDeactivationUseCase struct {
	userRepo           user.Repository
	deactRepo          user.DeactivationRequestRepository
	auditRepo          user.DeactivationAuditLogRepository
	emailSender        appshared.EmailSender
	sessionInvalidator user.SessionInvalidator
	txManager          appshared.TransactionManager
	refreshTokenRepo   user.RefreshTokenRepository
}

func NewConfirmDeactivationUseCase(
	userRepo user.Repository,
	deactRepo user.DeactivationRequestRepository,
	auditRepo user.DeactivationAuditLogRepository,
	emailSender appshared.EmailSender,
	sessionInvalidator user.SessionInvalidator,
	txManager appshared.TransactionManager,
	refreshTokenRepo user.RefreshTokenRepository,
) *ConfirmDeactivationUseCase {
	return &ConfirmDeactivationUseCase{
		userRepo:           userRepo,
		deactRepo:          deactRepo,
		auditRepo:          auditRepo,
		emailSender:        emailSender,
		sessionInvalidator: sessionInvalidator,
		txManager:          txManager,
		refreshTokenRepo:   refreshTokenRepo,
	}
}

func (uc *ConfirmDeactivationUseCase) Execute(ctx context.Context, input ConfirmDeactivationInput) (*ConfirmDeactivationOutput, error) {
	foundUser, err := uc.userRepo.FindByID(ctx, input.UserID)
	if err != nil {
		return nil, err
	}
	if foundUser == nil || foundUser.Status() == user.StatusDeactivated {
		return nil, apperror.NewConflict(user.ErrCodeAlreadyDeactivated, "User account is already deactivated or doesn't exist")
	}

	req, err := uc.deactRepo.FindPendingByUserID(ctx, input.UserID)
	if err != nil {
		return nil, err
	}
	if req == nil {
		return nil, apperror.NewNotFound(ErrCodeNoPendingRequest, "No pending deactivation request found")
	}

	now := time.Now()

	// Handle Blocked State
	if req.IsCurrentlyBlocked(now) {
		retryAfter := int(req.BlockedUntil().Sub(now).Seconds())
		if retryAfter < 0 {
			retryAfter = 0
		}
		return nil, apperror.NewTooManyRequests(ErrCodeMaxAttemptsExceeded, "Maximum confirmation attempts exceeded. Please try again later", retryAfter)
	}
	if req.IsBlocked() {
		// Block period has expired — invalidate the request entirely; user must start a new one
		req.MarkAsExpired(now)
		if err := uc.deactRepo.Update(ctx, req); err != nil {
			return nil, err
		}
		return nil, apperror.NewBadRequest(ErrCodeExpiredCode, "The confirmation code has expired. Please request a new one")
	}

	// Code expiration validation
	if req.IsExpired(now) {
		req.MarkAsExpired(now)
		if err := uc.deactRepo.Update(ctx, req); err != nil {
			return nil, err
		}
		return nil, apperror.NewBadRequest(ErrCodeExpiredCode, "The confirmation code has expired. Please request a new one")
	}

	// Code match validation
	if subtle.ConstantTimeCompare([]byte(req.VerificationCode()), []byte(input.Code)) != 1 {
		req.RegisterFailure(now)

		if req.IsBlocked() {
			if err := uc.deactRepo.Update(ctx, req); err != nil {
				return nil, err
			}
			return nil, apperror.NewTooManyRequests(ErrCodeMaxAttemptsExceeded, "Maximum confirmation attempts exceeded. Please try again later", 3600)
		}

		if err := uc.deactRepo.Update(ctx, req); err != nil {
			return nil, err
		}
		return nil, apperror.NewBadRequest(ErrCodeInvalidCode, "The confirmation code is invalid")
	}

	originalEmailStr := foundUser.Email().String()
	originalNicknameStr := foundUser.Nickname().String()

	anonSuffix := uuid.New().String()[:10]
	if err := foundUser.Deactivate(anonSuffix, now); err != nil {
		slog.ErrorContext(ctx, "failed to deactivate user domain object during confirmation", "user_id", input.UserID, "error", err)
		return nil, apperror.NewInternal()
	}

	req.Confirm(now)

	if err := uc.txManager.WithTx(ctx, func(txCtx context.Context) error {
		if err := uc.userRepo.Update(txCtx, foundUser); err != nil {
			return err
		}
		if err := uc.deactRepo.Update(txCtx, req); err != nil {
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
		// best-effort: account already deactivated, don't fail the operation.
		slog.ErrorContext(ctx, "failed to revoke refresh tokens after account deactivation", "user_id", foundUser.ID(), "error", err)
	}

	auditLogID := uuid.New().String()
	auditLog, err := user.NewDeactivationAuditLog(auditLogID, foundUser.ID(), originalEmailStr, originalNicknameStr, now, input.IP, input.UserAgent)
	if err != nil {
		slog.ErrorContext(ctx, "failed to build deactivation audit log", "user_id", foundUser.ID(), "error", err)
		return nil, apperror.NewInternal()
	}
	_ = uc.auditRepo.Save(ctx, auditLog)

	// Send final email
	if originalEmailStr != "" {
		_ = uc.emailSender.Send(ctx, appshared.EmailMessage{
			To:      originalEmailStr,
			Subject: "Account Deactivated",
			Body:    "Your account has been successfully deactivated based on your request. Your identity and email have been anonymized.",
			HTMLBody: emailtemplate.Wrap("Account Deactivated",
				"<p style=\"margin:0 0 12px;\">Your Training Center account has been successfully deactivated as requested.</p>"+
					"<p style=\"margin:0;color:#64748b;font-size:14px;\">Your identity and email address have been anonymized. If you believe this was done in error, please contact support.</p>"),
		})
	}

	return &ConfirmDeactivationOutput{SessionsInvalidated: sessionsInvalidated}, nil
}
