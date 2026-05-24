package user

import (
	"context"
	"time"

	"github.com/training-judge-center/backend/internal/domain/shared"
)

type TokenClaims struct {
	UserID   string
	Email    Email
	Role     shared.Role
	IssuedAt time.Time
}

type TokenService interface {
	GenerateToken(ctx context.Context, user *User) (string, error)
	ValidateToken(tokenString string) (*TokenClaims, error)
}
