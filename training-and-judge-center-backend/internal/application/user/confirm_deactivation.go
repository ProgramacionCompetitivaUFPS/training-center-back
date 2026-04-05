package user

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/training-judge-center/backend/internal/domain/notification"
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
	userRepo           user.UserRepository
	deactRepo          user.DeactivationRequestRepository
	auditRepo          user.DeactivationAuditLogRepository
	emailSender        notification.EmailSender
	sessionInvalidator user.SessionInvalidator
	txManager          user.TransactionManager
}

func NewConfirmDeactivationUseCase(
	userRepo user.UserRepository,
	deactRepo user.DeactivationRequestRepository,
	auditRepo user.DeactivationAuditLogRepository,
	emailSender notification.EmailSender,
	sessionInvalidator user.SessionInvalidator,
	txManager user.TransactionManager,
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
		return apperror.NewInternal()
	}
	if foundUser == nil || foundUser.Status() == user.StatusDeactivated {
		return apperror.NewConflict("ALREADY_DEACTIVATED", "User account is already deactivated or doesn't exist")
	}

	req, err := uc.deactRepo.FindPendingByUserID(ctx, input.UserID)
	if err != nil {
		return apperror.NewInternal()
	}
	if req == nil {
		return apperror.NewNotFound(apperror.ErrCodeNotFound, "No pending deactivation request found")
	}

	now := time.Now()

	// Handle Blocked State
	if req.IsBlocked() && req.BlockedUntil() != nil {
		if now.Before(*req.BlockedUntil()) {
			retryAfter := int(req.BlockedUntil().Sub(now).Seconds())
			if retryAfter < 0 {
				retryAfter = 0
			}
			return apperror.NewTooManyRequests("MAX_ATTEMPTS_EXCEEDED", "Maximum confirmation attempts exceeded. Please try again later", retryAfter)
		}
	}

	// Code expiration validation
	if req.IsExpired(now) {
		req.MarkAsExpired()
		if err := uc.deactRepo.Update(ctx, req); err != nil {
			return apperror.NewInternal()
		}
		return apperror.NewBadRequest("EXPIRED_CODE", "The confirmation code has expired. Please request a new one")
	}

	// Code match validation
	if req.VerificationCode() != input.Code {
		req.RegisterFailure()

		if req.IsBlocked() {
			if err := uc.deactRepo.Update(ctx, req); err != nil {
				return apperror.NewInternal()
			}
			return apperror.NewTooManyRequests("MAX_ATTEMPTS_EXCEEDED", "Maximum confirmation attempts exceeded. Please try again later", 3600)
		}

		if err := uc.deactRepo.Update(ctx, req); err != nil {
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
		return apperror.NewInternal()
	}

	if err := uc.sessionInvalidator.InvalidateAllUserSessions(ctx, foundUser.ID(), now); err != nil {
		slog.Error("failed to invalidate sessions after self-deactivation", "user_id", foundUser.ID(), "error", err)
	}

	auditLog := user.RestoreDeactivationAuditLog(uuid.NewString(), foundUser.ID(), originalEmailStr, originalNicknameStr, now, input.IP, input.UserAgent)
	if err := uc.auditRepo.Save(ctx, auditLog); err != nil {
		slog.Error("failed to save deactivation audit log", "user_id", foundUser.ID(), "error", err)
	}

	// Send final email
	if originalEmailStr != "" {
		if err := uc.emailSender.Send(ctx, notification.EmailMessage{
			To:      originalEmailStr,
			Subject: "Account Deactivated",
			Body:    "Your account has been successfully deactivated based on your request. Your identity and email have been anonymized.",
		}); err != nil {
			slog.Error("failed to send deactivation confirmation email", "error", err)
		}
	}

	return nil
}
