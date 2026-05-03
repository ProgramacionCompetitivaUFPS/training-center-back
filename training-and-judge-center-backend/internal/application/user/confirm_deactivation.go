package user

import (
	"context"
	"crypto/subtle"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/training-judge-center/backend/internal/application/shared"
	"github.com/training-judge-center/backend/internal/domain/user"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type ConfirmDeactivationInput struct {
	UserID    string
	Code      string
	IP        *string
	UserAgent *string
}

type ConfirmDeactivationUseCase struct {
	userRepo           user.Repository
	deactRepo          user.DeactivationRequestRepository
	auditRepo          user.DeactivationAuditLogRepository
	emailSender        shared.EmailSender
	sessionInvalidator user.SessionInvalidator
	txManager          TransactionManager
}

func NewConfirmDeactivationUseCase(
	userRepo user.Repository,
	deactRepo user.DeactivationRequestRepository,
	auditRepo user.DeactivationAuditLogRepository,
	emailSender shared.EmailSender,
	sessionInvalidator user.SessionInvalidator,
	txManager TransactionManager,
) *ConfirmDeactivationUseCase {
	return &ConfirmDeactivationUseCase{
		userRepo:           userRepo,
		deactRepo:          deactRepo,
		auditRepo:          auditRepo,
		emailSender:        emailSender,
		sessionInvalidator: sessionInvalidator,
		txManager:          txManager,
	}
}

func (uc *ConfirmDeactivationUseCase) Execute(ctx context.Context, input ConfirmDeactivationInput) error {
	foundUser, err := uc.userRepo.FindByID(ctx, input.UserID)
	if err != nil {
		slog.Error("failed to find user during deactivation confirmation", "user_id", input.UserID, "error", err)
		return apperror.NewInternal()
	}
	if foundUser == nil || foundUser.Status() == user.StatusDeactivated {
		return apperror.NewConflict("ALREADY_DEACTIVATED", "User account is already deactivated or doesn't exist")
	}

	req, err := uc.deactRepo.FindPendingByUserID(ctx, input.UserID)
	if err != nil {
		slog.Error("failed to find pending deactivation request", "user_id", input.UserID, "error", err)
		return apperror.NewInternal()
	}
	if req == nil {
		return apperror.NewNotFound(apperror.ErrCodeNotFound, "No pending deactivation request found")
	}

	now := time.Now()

	// Handle Blocked State
	if req.IsCurrentlyBlocked(now) {
		retryAfter := int(req.BlockedUntil().Sub(now).Seconds())
		if retryAfter < 0 {
			retryAfter = 0
		}
		return apperror.NewTooManyRequests("MAX_ATTEMPTS_EXCEEDED", "Maximum confirmation attempts exceeded. Please try again later", retryAfter)
	}
	if req.IsBlocked() {
		// Block period has expired — invalidate the request entirely; user must start a new one
		req.MarkAsExpired()
		if err := uc.deactRepo.Update(ctx, req); err != nil {
			slog.Error("failed to mark expired deactivation request", "user_id", input.UserID, "error", err)
			return apperror.NewInternal()
		}
		return apperror.NewBadRequest("EXPIRED_CODE", "The confirmation code has expired. Please request a new one")
	}

	// Code expiration validation
	if req.IsExpired(now) {
		req.MarkAsExpired()
		if err := uc.deactRepo.Update(ctx, req); err != nil {
			slog.Error("failed to mark expired deactivation request", "user_id", input.UserID, "error", err)
			return apperror.NewInternal()
		}
		return apperror.NewBadRequest("EXPIRED_CODE", "The confirmation code has expired. Please request a new one")
	}

	// Code match validation
	if subtle.ConstantTimeCompare([]byte(req.VerificationCode()), []byte(input.Code)) != 1 {
		req.RegisterFailure()

		if req.IsBlocked() {
			if err := uc.deactRepo.Update(ctx, req); err != nil {
				slog.Error("failed to persist blocked deactivation request", "user_id", input.UserID, "error", err)
				return apperror.NewInternal()
			}
			return apperror.NewTooManyRequests("MAX_ATTEMPTS_EXCEEDED", "Maximum confirmation attempts exceeded. Please try again later", 3600)
		}

		if err := uc.deactRepo.Update(ctx, req); err != nil {
			slog.Error("failed to persist failed deactivation attempt", "user_id", input.UserID, "error", err)
			return apperror.NewInternal()
		}
		return apperror.NewBadRequest("INVALID_CODE", "The confirmation code is invalid")
	}

	originalEmailStr := ""
	if foundUser.Email() != nil {
		originalEmailStr = foundUser.Email().String()
	}
	originalNicknameStr := foundUser.Nickname().String()

	if err := foundUser.Deactivate(); err != nil {
		slog.Error("failed to deactivate user domain object during confirmation", "user_id", input.UserID, "error", err)
		return apperror.NewInternal()
	}

	req.Confirm()

	if err := uc.txManager.WithTx(ctx, func(txCtx context.Context) error {
		if err := uc.userRepo.Update(txCtx, foundUser); err != nil {
			return err
		}
		if err := uc.deactRepo.Update(txCtx, req); err != nil {
			return err
		}
		return nil
	}); err != nil {
		slog.Error("failed to commit deactivation transaction", "user_id", input.UserID, "error", err)
		return apperror.NewInternal()
	}

	var sessionErr error
	if err := uc.sessionInvalidator.InvalidateAllUserSessions(ctx, foundUser.ID(), now); err != nil {
		slog.Error("failed to invalidate sessions after self-deactivation", "user_id", foundUser.ID(), "error", err)
		sessionErr = ErrSessionsNotInvalidated
	}

	auditLog := user.RestoreDeactivationAuditLog(uuid.NewString(), foundUser.ID(), originalEmailStr, originalNicknameStr, now, input.IP, input.UserAgent)
	if err := uc.auditRepo.Save(ctx, auditLog); err != nil {
		slog.Error("failed to save deactivation audit log", "user_id", foundUser.ID(), "error", err)
	}

	// Send final email
	if originalEmailStr != "" {
		if err := uc.emailSender.Send(ctx, shared.EmailMessage{
			To:      originalEmailStr,
			Subject: "Account Deactivated",
			Body:    "Your account has been successfully deactivated based on your request. Your identity and email have been anonymized.",
		}); err != nil {
			slog.Error("failed to send deactivation confirmation email", "error", err)
		}
	}

	return sessionErr
}
