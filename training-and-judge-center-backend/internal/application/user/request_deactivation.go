package user

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/training-judge-center/backend/internal/domain/notification"
	"github.com/training-judge-center/backend/internal/domain/user"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type RequestDeactivationInput struct {
	UserID string
}

type RequestDeactivationUseCase struct {
	userRepo        user.UserRepository
	deactRepo       user.DeactivationRequestRepository
	emailSender     notification.EmailSender
}

func NewRequestDeactivationUseCase(
	userRepo user.UserRepository,
	deactRepo user.DeactivationRequestRepository,
	emailSender notification.EmailSender,
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
		return apperror.NewInternal()
	}
	if foundUser == nil || foundUser.Status() == user.StatusDeactivated {
		return apperror.NewNotFound("NOT_FOUND", "User not found")
	}

	if foundUser.Role() == user.RoleAdmin {
		return apperror.NewForbidden("FORBIDDEN", "Administrators cannot deactivate their own account")
	}

	now := time.Now()

	// Invalidate previous requests to ensure only one code is active
	if err := uc.deactRepo.InvalidatePendingByUserID(ctx, foundUser.ID(), now); err != nil {
		return apperror.NewInternal()
	}

	code, err := generateSixDigitCode()
	if err != nil {
		return apperror.NewInternal()
	}

	req := user.RestoreDeactivationRequest(uuid.NewString(), foundUser.ID(), code, now.Add(15 * time.Minute), 0, nil, user.DeactivationStatusPending, now, now)

	if err := uc.deactRepo.Save(ctx, req); err != nil {
		return apperror.NewInternal()
	}

	if err := uc.emailSender.Send(ctx, notification.EmailMessage{
		To:      foundUser.Email().String(),
		Subject: "Account Deactivation Code",
		Body:    fmt.Sprintf("You requested to deactivate your account.\n\nYour confirmation code is: %s\n\nThis code will expire in 15 minutes. Note: Confirming this code will completely anonymize your account and log you out immediately.", code),
	}); err != nil {
		return apperror.NewServiceUnavailable("INTERNAL_SERVER_ERROR", "Failed to send verification email")
	}

	return nil
}
