package user

import (
	"context"

	"github.com/training-judge-center/backend/internal/domain/user"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type AdminDeactivateUserUseCase struct {
	repo user.UserRepository
}

func NewAdminDeactivateUserUseCase(repo user.UserRepository) *AdminDeactivateUserUseCase {
	return &AdminDeactivateUserUseCase{repo: repo}
}

func (uc *AdminDeactivateUserUseCase) Execute(ctx context.Context, requesterID, targetID string) error {
	if requesterID == targetID {
		return apperror.NewForbidden("CANNOT_SELF_DEACTIVATE", "Administrators cannot deactivate their own account")
	}

	foundUser, err := uc.repo.FindByID(ctx, targetID)
	if err != nil {
		return apperror.NewInternal()
	}
	if foundUser == nil {
		return apperror.NewNotFound("NOT_FOUND", "User not found")
	}

	if foundUser.Role == user.RoleAdmin {
		return apperror.NewForbidden("CANNOT_DEACTIVATE_ADMIN", "Cannot deactivate another administrator")
	}

	// Idempotent: already deactivated users return success immediately
	if foundUser.Status == user.StatusDeactivated {
		return nil
	}

	foundUser.Deactivate()

	if err := uc.repo.Update(ctx, foundUser); err != nil {
		return apperror.NewInternal()
	}

	return nil
}
