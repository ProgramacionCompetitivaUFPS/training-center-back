package user

import (
	"context"


	domain "github.com/training-judge-center/backend/internal/domain/user"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type GetMyProfileInput struct {
	UserID string
}

type GetMyProfileOutput struct {
	User          UserDTO
	IsFullProfile bool
}

type GetMyProfileUseCase struct {
	repo domain.Repository
}

func NewGetMyProfileUseCase(repo domain.Repository) *GetMyProfileUseCase {
	return &GetMyProfileUseCase{repo: repo}
}

func (uc *GetMyProfileUseCase) Execute(ctx context.Context, in GetMyProfileInput) (*GetMyProfileOutput, error) {
	foundUser, err := uc.repo.FindByID(ctx, in.UserID)
	if err != nil {
		return nil, err
	}
	if foundUser == nil {
		return nil, apperror.NewNotFound(domain.ErrCodeUserNotFound, "User not found")
	}
	return &GetMyProfileOutput{User: userToDTO(foundUser), IsFullProfile: true}, nil
}
