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
	GoogleLinked  bool
	HasPassword   bool
}

type GetMyProfileUseCase struct {
	repo              domain.Repository
	oauthIdentityRepo domain.OAuthIdentityRepository
}

func NewGetMyProfileUseCase(repo domain.Repository, oauthIdentityRepo domain.OAuthIdentityRepository) *GetMyProfileUseCase {
	return &GetMyProfileUseCase{repo: repo, oauthIdentityRepo: oauthIdentityRepo}
}

func (uc *GetMyProfileUseCase) Execute(ctx context.Context, in GetMyProfileInput) (*GetMyProfileOutput, error) {
	foundUser, err := uc.repo.FindByID(ctx, in.UserID)
	if err != nil {
		return nil, err
	}
	if foundUser == nil {
		return nil, apperror.NewNotFound(domain.ErrCodeUserNotFound, "User not found")
	}

	identity, err := uc.oauthIdentityRepo.FindByUserID(ctx, foundUser.ID(), domain.OAuthProviderGoogle)
	if err != nil {
		return nil, err
	}

	return &GetMyProfileOutput{
		User:          userToDTO(foundUser),
		IsFullProfile: true,
		GoogleLinked:  identity != nil,
		HasPassword:   foundUser.Password().HasPassword(),
	}, nil
}
