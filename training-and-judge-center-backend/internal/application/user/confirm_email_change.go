package user

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/training-judge-center/backend/internal/domain/notification"
	"github.com/training-judge-center/backend/internal/domain/user"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type ConfirmEmailChangeInput struct {
	UserID string
	Code   string
}

type ConfirmEmailChangeUseCase struct {
	userRepo        user.UserRepository
	emailChangeRepo user.EmailChangeRepository
	emailSender     notification.EmailSender
}

func NewConfirmEmailChangeUseCase(
	userRepo user.UserRepository,
	emailChangeRepo user.EmailChangeRepository,
	emailSender notification.EmailSender,
) *ConfirmEmailChangeUseCase {
	return &ConfirmEmailChangeUseCase{
		userRepo:        userRepo,
		emailChangeRepo: emailChangeRepo,
		emailSender:     emailSender,
	}
}

func (uc *ConfirmEmailChangeUseCase) Execute(ctx context.Context, input ConfirmEmailChangeInput) (*user.Email, error) {
	u, err := uc.userRepo.FindByID(ctx, input.UserID)
	if err != nil {
		return nil, apperror.NewInternal()
	}
	if u == nil {
		return nil, apperror.NewUnauthorized("INVALID_CREDENTIALS", "Invalid credentials")
	}

	req, err := uc.emailChangeRepo.FindByCodeAndUserID(ctx, input.Code, input.UserID)
	if err != nil {
		return nil, apperror.NewInternal()
	}
	
	if req == nil {
		return nil, apperror.NewBadRequest("INVALID_CODE", "The verification code is invalid or has expired")
	}

	if req.Status != user.StatusPending || req.IsExpired(time.Now()) {
		return nil, apperror.NewBadRequest("INVALID_CODE", "The verification code is invalid or has expired")
	}

	emailExists, err := uc.userRepo.ExistsByEmail(ctx, req.NewEmail)
	if err != nil {
		return nil, apperror.NewInternal()
	}
	if emailExists {
		return nil, apperror.NewConflict("EMAIL_ALREADY_EXISTS", "The email address is already in use")
	}

	oldEmail := u.Email.String()

	req.MarkAsUsed(time.Now())
	if err := uc.emailChangeRepo.Update(ctx, req); err != nil {
		return nil, apperror.NewInternal()
	}

	u.Update(nil, nil, nil, &req.NewEmail, nil)
	if err := uc.userRepo.Update(ctx, u); err != nil {
		return nil, apperror.NewInternal()
	}

	if err := uc.emailSender.Send(ctx, notification.EmailMessage{
		To:      oldEmail,
		Subject: "Security Alert: Your Email Was Changed",
		Body:    "Your account email has been successfully changed to a new one. If you did not make this change, please contact support immediately.",
	}); err != nil {
		slog.Error("failed to send security alert to old email", "email", oldEmail, "error", err)
	}
	
	if err := uc.emailSender.Send(ctx, notification.EmailMessage{
		To:      req.NewEmail.String(),
		Subject: "Email successfully updated",
		Body:    fmt.Sprintf("Hello %s, your email address has been successfully verified and updated on our platform.", u.Name),
	}); err != nil {
		slog.Error("failed to send confirmation to new email", "email", req.NewEmail.String(), "error", err)
	}

	return &req.NewEmail, nil
}
