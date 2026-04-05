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
}

func NewConfirmDeactivationUseCase(
	userRepo user.UserRepository,
	deactRepo user.DeactivationRequestRepository,
	auditRepo user.DeactivationAuditLogRepository,
	emailSender notification.EmailSender,
	sessionInvalidator user.SessionInvalidator,
) *ConfirmDeactivationUseCase {
	return &ConfirmDeactivationUseCase{
		userRepo:           userRepo,
		deactRepo:          deactRepo,
		auditRepo:          auditRepo,
		emailSender:        emailSender,
		sessionInvalidator: sessionInvalidator,
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
	if now.After(req.ExpiresAt()) {
		req.MarkAsExpired()
		_ = uc.deactRepo.Update(ctx, req)
		return apperror.NewBadRequest("EXPIRED_CODE", "The confirmation code has expired. Please request a new one")
	}

	// Code match validation
	if req.VerificationCode() != input.Code {
		req.RegisterFailure()

		if req.IsBlocked() {
			_ = uc.deactRepo.Update(ctx, req)
			return apperror.NewTooManyRequests("MAX_ATTEMPTS_EXCEEDED", "Maximum confirmation attempts exceeded. Please try again later", 3600)
		}

		_ = uc.deactRepo.Update(ctx, req)
		return apperror.NewBadRequest("INVALID_CODE", "The confirmation code is invalid")
	}

	// Execution: Deactivate
	originalEmailStr := ""
	if foundUser.Email() != nil {
		originalEmailStr = foundUser.Email().String()
	}
	originalNicknameStr := foundUser.Nickname().String()

	foundUser.Deactivate() // Applies StatusDeactivated, Email=nil, Anon Nickname, timestamps

	if err := uc.userRepo.Update(ctx, foundUser); err != nil {
		return apperror.NewInternal()
	}

	// Close request
	req.Confirm()
	if err := uc.deactRepo.Update(ctx, req); err != nil {
		return apperror.NewInternal()
	}

	// Invalidate Sessions
	if err := uc.sessionInvalidator.InvalidateAllUserSessions(ctx, foundUser.ID(), now); err != nil {
		slog.Error("failed to invalidate sessions after self-deactivation", "user_id", foundUser.ID(), "error", err)
	}

	// Audit Log
	auditLog := user.RestoreDeactivationAuditLog(uuid.NewString(), foundUser.ID(), originalEmailStr, originalNicknameStr, now, input.IP, input.UserAgent)
	_ = uc.auditRepo.Save(ctx, auditLog) // fail-safe (ignore error if it fails instead of reverting user)

	// Send final email
	if originalEmailStr != "" {
		_ = uc.emailSender.Send(ctx, notification.EmailMessage{
			To:      originalEmailStr,
			Subject: "Account Deactivated",
			Body:    "Your account has been successfully deactivated based on your request. Your identity and email have been anonymized.",
		})
	}

	return nil
}
