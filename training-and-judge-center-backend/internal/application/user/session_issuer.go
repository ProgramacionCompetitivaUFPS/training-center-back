package user

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/training-judge-center/backend/internal/domain/user"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type issuedSession struct {
	Token            string
	RefreshToken     string
	SessionExpiresAt time.Time
}

func issueSession(
	ctx context.Context,
	foundUser *user.User,
	rememberSession bool,
	userAgent, ipAddress *string,
	tokenService user.TokenService,
	refreshTokenRepo user.RefreshTokenRepository,
	refreshTokenCodec RefreshTokenCodec,
	now time.Time,
) (*issuedSession, error) {
	sessionID := uuid.New().String()

	token, err := tokenService.GenerateToken(ctx, foundUser, sessionID)
	if err != nil {
		return nil, err // nothing persisted yet — no orphaned refresh token
	}

	refreshSecret, err := generateRefreshTokenSecret()
	if err != nil {
		return nil, apperror.NewInternal()
	}

	newRefreshToken, err := user.NewRefreshToken(
		uuid.New().String(),
		foundUser.ID(),
		sessionID,
		hashRefreshTokenSecret(refreshSecret),
		userAgent,
		ipAddress,
		rememberSession,
		now,
	)
	if err != nil {
		return nil, err
	}

	wrapped, err := refreshTokenCodec.Wrap(ctx, refreshSecret, foundUser.ID())
	if err != nil {
		return nil, err
	}

	if err := refreshTokenRepo.Save(ctx, newRefreshToken); err != nil {
		return nil, err
	}

	return &issuedSession{
		Token:            token,
		RefreshToken:     wrapped,
		SessionExpiresAt: newRefreshToken.AbsoluteExpiresAt(),
	}, nil
}
