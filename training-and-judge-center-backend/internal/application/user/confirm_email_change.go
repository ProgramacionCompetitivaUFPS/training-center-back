package user

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/training-judge-center/backend/internal/application/shared"
	"github.com/training-judge-center/backend/internal/domain/user"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type ConfirmEmailChangeInput struct {
	UserID string
	Code   string
}

type ConfirmEmailChangeUseCase struct {
	userRepo        user.Repository
	emailChangeRepo user.EmailChangeRepository
	emailSender     shared.EmailSender
	txManager       TransactionManager
}

func NewConfirmEmailChangeUseCase(
	userRepo user.Repository,
	emailChangeRepo user.EmailChangeRepository,
	emailSender shared.EmailSender,
	txManager TransactionManager,
) *ConfirmEmailChangeUseCase {
	return &ConfirmEmailChangeUseCase{
		userRepo:        userRepo,
		emailChangeRepo: emailChangeRepo,
		emailSender:     emailSender,
		txManager:       txManager,
	}
}

func (uc *ConfirmEmailChangeUseCase) Execute(ctx context.Context, input ConfirmEmailChangeInput) (*user.Email, error) {
	u, err := uc.userRepo.FindByID(ctx, input.UserID)
	if err != nil {
		slog.Error("failed to find user during email change confirmation", "user_id", input.UserID, "error", err)
		return nil, apperror.NewInternal()
	}
	if u == nil {
		return nil, apperror.NewUnauthorized("INVALID_CREDENTIALS", "Invalid credentials")
	}

	req, err := uc.emailChangeRepo.FindByCodeAndUserID(ctx, input.Code, input.UserID)
	if err != nil {
		slog.Error("failed to find email change request by code", "user_id", input.UserID, "error", err)
		return nil, apperror.NewInternal()
	}
	
	if req == nil {
		return nil, apperror.NewBadRequest("INVALID_CODE", "The verification code is invalid or has expired")
	}

	now := time.Now()
	if req.Status() != user.StatusPending || req.IsExpired(now) {
		return nil, apperror.NewBadRequest("INVALID_CODE", "The verification code is invalid or has expired")
	}

	oldEmail := u.Email().String()
	newEmailVal := req.NewEmail()
	if err := u.UpdateEmail(newEmailVal, now); err != nil {
		slog.Error("failed to update email on user domain object", "user_id", input.UserID, "error", err)
		return nil, apperror.NewInternal()
	}
	if err := req.MarkAsUsed(now); err != nil {
		slog.Error("failed to mark email change request as used", "user_id", input.UserID, "error", err)
		return nil, apperror.NewInternal()
	}

	if err := uc.txManager.WithTx(ctx, func(txCtx context.Context) error {
		if err := uc.userRepo.Update(txCtx, u); err != nil {
			return err
		}
		if err := uc.emailChangeRepo.Update(txCtx, req); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return nil, err
	}

	if err := uc.emailSender.Send(ctx, shared.EmailMessage{
		To:      oldEmail,
		Subject: "Security Alert: Your Email Was Changed",
		Body:    "Your account email has been successfully changed to a new one. If you did not make this change, please contact support immediately.",
	}); err != nil {
		slog.Error("failed to send security alert to old email", "email", oldEmail, "error", err)
	}
	
	if err := uc.emailSender.Send(ctx, shared.EmailMessage{
		To:      req.NewEmail().String(),
		Subject: "Email successfully updated",
		Body:    fmt.Sprintf("Hello %s, your email address has been successfully verified and updated on our platform.", u.Name()),
	}); err != nil {
		slog.Error("failed to send confirmation to new email", "email", req.NewEmail().String(), "error", err)
	}

	return &newEmailVal, nil
}
