package user

import (
	"context"
	"time"

	"github.com/training-judge-center/backend/internal/domain/user"
)

type LogoutInput struct {
	RefreshToken string // valor envuelto de la cookie (RefreshTokenCodec.Wrap); "" si no había cookie
}

type LogoutUseCase struct {
	refreshTokenRepo   user.RefreshTokenRepository
	sessionInvalidator user.SessionInvalidator
	refreshTokenCodec  RefreshTokenCodec
}

func NewLogoutUseCase(refreshTokenRepo user.RefreshTokenRepository, sessionInvalidator user.SessionInvalidator, refreshTokenCodec RefreshTokenCodec) *LogoutUseCase {
	return &LogoutUseCase{
		refreshTokenRepo:   refreshTokenRepo,
		sessionInvalidator: sessionInvalidator,
		refreshTokenCodec:  refreshTokenCodec,
	}
}

func (uc *LogoutUseCase) Execute(ctx context.Context, in LogoutInput) error {
	if in.RefreshToken == "" {
		return nil
	}

	plainSecret, _, err := uc.refreshTokenCodec.Unwrap(in.RefreshToken)
	if err != nil {
		return nil
	}

	tokenHash := hashRefreshTokenSecret(plainSecret)
	token, err := uc.refreshTokenRepo.FindByTokenHash(ctx, tokenHash)
	if err != nil {
		return err
	}
	if token == nil || token.IsRevoked() {
		return nil
	}

	now := time.Now()

	if err := uc.sessionInvalidator.InvalidateSession(ctx, token.FamilyID(), now); err != nil {
		return err
	}
	if err := uc.refreshTokenRepo.RevokeByFamilyID(ctx, token.FamilyID(), now); err != nil {
		return err
	}
	return nil
}
