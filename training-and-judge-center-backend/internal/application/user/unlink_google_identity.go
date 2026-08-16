package user

import (
	"context"

	"github.com/training-judge-center/backend/internal/domain/user"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type UnlinkGoogleIdentityInput struct {
	UserID string
}

type UnlinkGoogleIdentityUseCase struct {
	oauthIdentityRepo user.OAuthIdentityRepository
}

func NewUnlinkGoogleIdentityUseCase(oauthIdentityRepo user.OAuthIdentityRepository) *UnlinkGoogleIdentityUseCase {
	return &UnlinkGoogleIdentityUseCase{oauthIdentityRepo: oauthIdentityRepo}
}

func (uc *UnlinkGoogleIdentityUseCase) Execute(ctx context.Context, in UnlinkGoogleIdentityInput) error {
	identity, err := uc.oauthIdentityRepo.FindByUserID(ctx, in.UserID)
	if err != nil {
		return err
	}
	if identity == nil {
		return apperror.NewNotFound(user.ErrCodeOAuthIdentityNotFound, "no linked Google account found for this user")
	}
	return uc.oauthIdentityRepo.DeleteByUserID(ctx, in.UserID)
}
