package user

import (
	"context"
	"log/slog"

	domain "github.com/training-judge-center/backend/internal/domain/user"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type GetMyProfileInput struct {
	UserID string
}

type GetMyProfileUseCase struct {
	repo domain.Repository
}

func NewGetMyProfileUseCase(repo domain.Repository) *GetMyProfileUseCase {
	return &GetMyProfileUseCase{repo: repo}
}

func (uc *GetMyProfileUseCase) Execute(ctx context.Context, in GetMyProfileInput) (*UserProfileOutput, error) {
	foundUser, err := uc.repo.FindByID(ctx, in.UserID)
	if err != nil {
		slog.ErrorContext(ctx, "failed to find user profile by id", "user_id", in.UserID, "error", err)
		return nil, apperror.NewInternal()
	}
	if foundUser == nil {
		return nil, apperror.NewNotFound("NOT_FOUND", "User not found")
	}
	return &UserProfileOutput{User: userToDTO(foundUser), IsFullProfile: true}, nil
}
