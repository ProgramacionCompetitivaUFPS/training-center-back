package user

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
	appshared "github.com/training-judge-center/backend/internal/application/shared"
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
	refreshTokenRepo   user.RefreshTokenRepository
	txManager          appshared.TransactionManager
	teamCleaner        TeamMembershipCleaner
}

func NewAdminDeactivateUserUseCase(repo user.Repository, sessionInvalidator user.SessionInvalidator, refreshTokenRepo user.RefreshTokenRepository, txManager appshared.TransactionManager, teamCleaner TeamMembershipCleaner) *AdminDeactivateUserUseCase {
	return &AdminDeactivateUserUseCase{
		repo:               repo,
		sessionInvalidator: sessionInvalidator,
		refreshTokenRepo:   refreshTokenRepo,
		txManager:          txManager,
		teamCleaner:        teamCleaner,
	}
}

func (uc *AdminDeactivateUserUseCase) Execute(ctx context.Context, input AdminDeactivateUserInput) error {
	if input.RequesterID == input.TargetID {
		return apperror.NewForbidden(ErrCodeCannotSelfDeactivate, "Administrators cannot deactivate their own account")
	}

	foundUser, err := uc.repo.FindByID(ctx, input.TargetID)
	if err != nil {
		return err
	}
	if foundUser == nil {
		return apperror.NewNotFound(user.ErrCodeUserNotFound, "User not found")
	}

	if foundUser.Role() == shared.RoleAdmin {
		return apperror.NewForbidden(ErrCodeCannotDeactivateAdmin, "Cannot deactivate another administrator")
	}

	// Idempotent: already deactivated users return success immediately
	if foundUser.Status() == user.StatusDeactivated {
		return nil
	}

	now := time.Now()
	anonSuffix := uuid.New().String()[:10]
	if err := foundUser.Deactivate(anonSuffix, now); err != nil {
		slog.ErrorContext(ctx, "failed to deactivate user domain object", "user_id", foundUser.ID(), "error", err)
		return apperror.NewInternal()
	}

	// Invalidate sessions and revoke refresh tokens BEFORE persisting to DB: if either
	// fails, the DB write never happens and the caller receives an error with clean state.
	// If DB write fails after this point, the user loses their active session
	// but is not officially deactivated — a minor inconsistency, not a security risk.
	if err := uc.sessionInvalidator.InvalidateAllUserSessions(ctx, foundUser.ID(), now); err != nil {
		return err
	}
	if err := uc.refreshTokenRepo.RevokeAllByUserID(ctx, foundUser.ID(), now); err != nil {
		return err
	}

	// The account and its team memberships change together: a person who can no longer sign in
	// could never be taken off a team by anyone else.
	return uc.txManager.WithTx(ctx, func(txCtx context.Context) error {
		if err := uc.repo.Update(txCtx, foundUser); err != nil {
			return err
		}
		return uc.teamCleaner.RemoveUserFromAllTeams(txCtx, foundUser.ID())
	})
}
