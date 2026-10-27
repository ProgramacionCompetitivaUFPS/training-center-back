package auth

import (
	"context"
	"log/slog"

	"github.com/golang-jwt/jwt/v5"
	appuser "github.com/training-judge-center/backend/internal/application/user"
	"github.com/training-judge-center/backend/pkg/apperror"
)

var _ appuser.RefreshTokenCodec = (*JWTRefreshTokenCodec)(nil)

type refreshTokenClaims struct {
	Secret               string `json:"secret"`
	jwt.RegisteredClaims        // Subject = userID
}

type JWTRefreshTokenCodec struct {
	secret []byte
}

func NewJWTRefreshTokenCodec(secret string) *JWTRefreshTokenCodec {
	return &JWTRefreshTokenCodec{secret: []byte(secret)}
}

func (c *JWTRefreshTokenCodec) Wrap(ctx context.Context, secret, userID string) (string, error) {
	claims := refreshTokenClaims{
		Secret:           secret,
		RegisteredClaims: jwt.RegisteredClaims{Subject: userID},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(c.secret)
	if err != nil {
		slog.ErrorContext(ctx, "failed to sign refresh token envelope", "user_id", userID, "error", err)
		return "", apperror.NewInternal()
	}
	return signed, nil
}

func (c *JWTRefreshTokenCodec) Unwrap(wrapped string) (secret, userID string, err error) {
	claims := &refreshTokenClaims{}
	_, err = jwt.ParseWithClaims(wrapped, claims, func(*jwt.Token) (any, error) {
		return c.secret, nil
	}, jwt.WithValidMethods([]string{"HS256"}))
	if err != nil || claims.Secret == "" || claims.Subject == "" {
		return "", "", apperror.NewUnauthorized(apperror.ErrCodeUnauthorized, "invalid session")
	}
	return claims.Secret, claims.Subject, nil
}
