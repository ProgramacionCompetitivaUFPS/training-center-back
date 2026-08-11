package user

import (
	"context"
	"time"

	"github.com/training-judge-center/backend/internal/domain/shared"
)

type TokenClaims struct {
	UserID    string
	Email     Email
	Role      shared.Role
	IssuedAt  time.Time
	SessionID string
}

type TokenService interface {
	GenerateToken(ctx context.Context, user *User, sessionID string) (string, error)
	ValidateToken(tokenString string) (*TokenClaims, error)
}
