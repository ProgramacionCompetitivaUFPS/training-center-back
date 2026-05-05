package user

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/training-judge-center/backend/internal/domain/shared"
	"github.com/training-judge-center/backend/internal/domain/user"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type AdminDeactivateUserInput struct {
	RequesterID string
	TargetID    string
}

type AdminDeactivateUserUseCase struct {
	repo               user.Repository
	sessionInvalidator user.SessionInvalidator
}

func NewAdminDeactivateUserUseCase(repo user.Repository, sessionInvalidator user.SessionInvalidator) *AdminDeactivateUserUseCase {
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
		slog.Error("failed to find user by id during admin deactivation", "target_id", input.TargetID, "error", err)
		return apperror.NewInternal()
	}
	if foundUser == nil {
		return apperror.NewNotFound(apperror.ErrCodeNotFound, "User not found")
	}

	if foundUser.Role() == shared.RoleAdmin {
		return apperror.NewForbidden(user.ErrCodeCannotDeactivateAdmin, "Cannot deactivate another administrator")
	}

	// Idempotent: already deactivated users return success immediately
	if foundUser.Status() == user.StatusDeactivated {
		return nil
	}

	now := time.Now()
	anonSuffix := uuid.New().String()[:10]
	if err := foundUser.Deactivate(anonSuffix, now); err != nil {
		slog.Error("failed to deactivate user domain object", "user_id", foundUser.ID(), "error", err)
		return apperror.NewInternal()
	}

	// Invalidate sessions BEFORE persisting to DB: if Redis fails, the DB
	// write never happens and the caller receives an error with clean state.
	// If DB write fails after this point, the user loses their active session
	// but is not officially deactivated — a minor inconsistency, not a security risk.
	if err := uc.sessionInvalidator.InvalidateAllUserSessions(ctx, foundUser.ID(), now); err != nil {
		slog.Error("failed to invalidate sessions during admin deactivation", "user_id", foundUser.ID(), "error", err)
		return apperror.NewInternal()
	}

	if err := uc.repo.Update(ctx, foundUser); err != nil {
		slog.Error("failed to persist user deactivation", "user_id", foundUser.ID(), "error", err)
		return apperror.NewInternal()
	}

	return nil
}
