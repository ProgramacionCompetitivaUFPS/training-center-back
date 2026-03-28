package user

import (
	"context"
	"log/slog"
	"time"

	"github.com/training-judge-center/backend/internal/domain/user"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type AdminDeactivateUserInput struct {
	RequesterID string
	TargetID    string
}

type AdminDeactivateUserUseCase struct {
	repo               user.UserRepository
	sessionInvalidator user.SessionInvalidator
}

func NewAdminDeactivateUserUseCase(repo user.UserRepository, sessionInvalidator user.SessionInvalidator) *AdminDeactivateUserUseCase {
	return &AdminDeactivateUserUseCase{
		repo:               repo,
		sessionInvalidator: sessionInvalidator,
	}
}

func (uc *AdminDeactivateUserUseCase) Execute(ctx context.Context, input AdminDeactivateUserInput) error {
	if input.RequesterID == input.TargetID {
		return apperror.NewForbidden(user.ErrCodeCannotSelfDeactivate, "Administrators cannot deactivate their own account")
	}

	foundUser, err := uc.repo.FindByID(ctx, input.TargetID)
	if err != nil {
		return apperror.NewInternal()
	}
	if foundUser == nil {
		return apperror.NewNotFound(apperror.ErrCodeNotFound, "User not found")
	}

	if foundUser.Role == user.RoleAdmin {
		return apperror.NewForbidden(user.ErrCodeCannotDeactivateAdmin, "Cannot deactivate another administrator")
	}

	// Idempotent: already deactivated users return success immediately
	if foundUser.Status == user.StatusDeactivated {
		return nil
	}

	foundUser.Deactivate()

	if err := uc.repo.Update(ctx, foundUser); err != nil {
		return apperror.NewInternal()
	}

	now := time.Now()
	if err := uc.sessionInvalidator.InvalidateAllUserSessions(ctx, foundUser.ID, now); err != nil {
		slog.Error("failed to invalidate sessions after admin deactivation", "user_id", foundUser.ID, "error", err)
	}

	return nil
}
