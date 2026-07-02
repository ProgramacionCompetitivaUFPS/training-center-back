package problem

import (
	"context"

	"github.com/training-judge-center/backend/internal/domain/shared"
	"github.com/training-judge-center/backend/pkg/apperror"
)

func resolveModifierID(ctx context.Context, userProvider UserProvider, nickname string) (shared.UserID, error) {
	userID, found, err := userProvider.GetIDByNickname(ctx, nickname)
	if err != nil {
		return shared.UserID{}, err
	}
	if !found {
		return shared.UserID{}, apperror.NewNotFound(ErrCodeUserNotFound, "User not found")
	}
	return shared.NewUserID(userID)
}
