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
	if foundUser == nil || foundUser.Status == user.StatusDeactivated {
		return &apperror.AppError{
			Code:       "ALREADY_DEACTIVATED",
			Message:    "User account is already deactivated or doesn't exist",
			StatusCode: 409,
		}
	}

	req, err := uc.deactRepo.FindPendingByUserID(ctx, input.UserID)
	if err != nil {
		return apperror.NewInternal()
	}
	if req == nil {
		return &apperror.AppError{
			Code:       "NOT_FOUND",
			Message:    "No pending deactivation request found",
			StatusCode: 404,
		}
	}

	now := time.Now()

	// Handle Blocked State
	if req.Status == user.DeactivationStatusBlocked && req.BlockedUntil != nil {
		if now.Before(*req.BlockedUntil) {
			return &apperror.AppError{
				Code:       "MAX_ATTEMPTS_EXCEEDED",
				Message:    "Maximum confirmation attempts exceeded. Please try again later",
				StatusCode: 429,
			}
		}
		// Block expired. Let them try again, but ideally they need a new code if it expired over an hour.
		// Wait, FR-012 says requesting a new code doesn't reset attempts, but what happens when 1 hour passes?
		// They can request a new code. But here they are trying to *confirm* the old code? 
		// Actually, if the 1-hour block is over, the old code is mathematically expired anyway (15m is less than 1h). So it will hit EXPIRED_CODE below.
	}

	// Code expiration validation
	if now.After(req.ExpiresAt) {
		req.Status = user.DeactivationStatusExpired
		req.UpdatedAt = now
		_ = uc.deactRepo.Update(ctx, req)
		return &apperror.AppError{
			Code:       "EXPIRED_CODE",
			Message:    "The confirmation code has expired. Please request a new one",
			StatusCode: 400,
		}
	}

	// Code match validation
	if req.VerificationCode != input.Code {
		req.Attempts++
		req.UpdatedAt = now
		if req.Attempts >= 5 {
			req.Status = user.DeactivationStatusBlocked
			blockedUntil := now.Add(time.Hour)
			req.BlockedUntil = &blockedUntil
			_ = uc.deactRepo.Update(ctx, req)
			return &apperror.AppError{
				Code:       "MAX_ATTEMPTS_EXCEEDED",
				Message:    "Maximum confirmation attempts exceeded. Please try again later",
				StatusCode: 429,
			}
		}

		_ = uc.deactRepo.Update(ctx, req)
		return &apperror.AppError{
			Code:       "INVALID_CODE",
			Message:    "The confirmation code is invalid",
			StatusCode: 400,
		}
	}

	// Execution: Deactivate
	originalEmailStr := ""
	if foundUser.Email != nil {
		originalEmailStr = foundUser.Email.String()
	}
	originalNicknameStr := foundUser.Nickname.String()

	foundUser.Deactivate() // Applies StatusDeactivated, Email=nil, Anon Nickname, timestamps

	if err := uc.userRepo.Update(ctx, foundUser); err != nil {
		return apperror.NewInternal()
	}

	// Close request
	req.Status = user.DeactivationStatusConfirmed
	req.UpdatedAt = now
	if err := uc.deactRepo.Update(ctx, req); err != nil {
		return apperror.NewInternal()
	}

	// Invalidate Sessions
	if err := uc.sessionInvalidator.InvalidateAllUserSessions(ctx, foundUser.ID, now); err != nil {
		slog.Error("failed to invalidate sessions after self-deactivation", "user_id", foundUser.ID, "error", err)
	}

	// Audit Log
	auditLog := &user.DeactivationAuditLog{
		ID:               uuid.NewString(),
		UserID:           foundUser.ID,
		OriginalEmail:    originalEmailStr,
		OriginalNickname: originalNicknameStr,
		OccurredAt:       now,
		IP:               input.IP,
		UserAgent:        input.UserAgent,
	}
	_ = uc.auditRepo.Save(ctx, auditLog) // fail-safe (ignore error if it fails instead of reverting user)

	// Send final email
	if originalEmailStr != "" {
		body := "Your account has been successfully deactivated based on your request. Your identity and email have been anonymized."
		_ = uc.emailSender.Send(ctx, originalEmailStr, "Account Deactivated", body)
	}

	return nil
}
