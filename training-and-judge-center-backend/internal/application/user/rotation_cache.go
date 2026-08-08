package user

import (
	"context"
	"time"
)

// RefreshOutput es el resultado de una rotación exitosa (o de un replay servido desde el
// cache). Se define acá, no en refresh.go, porque RotationCache lo necesita antes de que
// RefreshUseCase exista.
type RefreshOutput struct {
	Token            string
	RefreshToken     string
	SessionExpiresAt time.Time
}

type RotationCache interface {
	Save(ctx context.Context, oldTokenHash string, output RefreshOutput, ttl time.Duration) error
	Get(ctx context.Context, oldTokenHash string) (*RefreshOutput, error)
}
