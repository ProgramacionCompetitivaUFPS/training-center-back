package user

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/training-judge-center/backend/internal/application/shared"
	domainShared "github.com/training-judge-center/backend/internal/domain/shared"
	"github.com/training-judge-center/backend/internal/domain/user"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type RequestDeactivationInput struct {
	UserID string
}

type RequestDeactivationUseCase struct {
	userRepo        user.UserRepository
	deactRepo       user.DeactivationRequestRepository
	emailSender     shared.EmailSender
}

func NewRequestDeactivationUseCase(
	userRepo user.UserRepository,
	deactRepo user.DeactivationRequestRepository,
	emailSender shared.EmailSender,
) *RequestDeactivationUseCase {
	return &RequestDeactivationUseCase{
		userRepo:    userRepo,
		deactRepo:   deactRepo,
		emailSender: emailSender,
	}
}

func (uc *RequestDeactivationUseCase) Execute(ctx context.Context, input RequestDeactivationInput) error {
	foundUser, err := uc.userRepo.FindByID(ctx, input.UserID)
	if err != nil {
		slog.Error("failed to find user during deactivation request", "user_id", input.UserID, "error", err)
		return apperror.NewInternal()
	}
	if foundUser == nil || foundUser.Status() == user.StatusDeactivated {
		return apperror.NewNotFound("NOT_FOUND", "User not found")
	}

	if foundUser.Role() == domainShared.RoleAdmin {
		return apperror.NewForbidden("FORBIDDEN", "Administrators cannot deactivate their own account")
	}

	now := time.Now()

	// Invalidate previous requests to ensure only one code is active
	if err := uc.deactRepo.InvalidatePendingByUserID(ctx, foundUser.ID(), now); err != nil {
		slog.Error("failed to invalidate pending deactivation requests", "user_id", foundUser.ID(), "error", err)
		return apperror.NewInternal()
	}

	code, err := generateSixDigitCode()
	if err != nil {
		slog.Error("failed to generate deactivation code", "user_id", foundUser.ID(), "error", err)
		return apperror.NewInternal()
	}

	req := user.RestoreDeactivationRequest(uuid.NewString(), foundUser.ID(), code, now.Add(15*time.Minute), 0, nil, user.DeactivationStatusPending, now, now)

	if err := uc.deactRepo.Save(ctx, req); err != nil {
		slog.Error("failed to save deactivation request", "user_id", foundUser.ID(), "error", err)
		return apperror.NewInternal()
	}

	if err := uc.emailSender.Send(ctx, shared.EmailMessage{
		To:      foundUser.Email().String(),
		Subject: "Account Deactivation Code",
		Body:    fmt.Sprintf("You requested to deactivate your account.\n\nYour confirmation code is: %s\n\nThis code will expire in 15 minutes. Note: Confirming this code will completely anonymize your account and log you out immediately.", code),
	}); err != nil {
		slog.Error("failed to send deactivation code email", "user_id", foundUser.ID(), "error", err)
		return apperror.NewServiceUnavailable("EMAIL_DELIVERY_FAILED", "Failed to send verification email")
	}

	return nil
}
