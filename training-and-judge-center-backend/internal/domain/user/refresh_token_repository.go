package user

import (
	"context"
	"time"
)

type RefreshTokenRepository interface {
	Save(ctx context.Context, token *RefreshToken) error
	FindByTokenHash(ctx context.Context, tokenHash string) (*RefreshToken, error)
	FindActiveByFamilyID(ctx context.Context, familyID string) (*RefreshToken, error)

	// Rotate atomically revokes oldTokenHash and persists newToken as its successor.
	// rotated is false if oldTokenHash was already revoked — callers must not treat that
	// as success (reuse or a concurrent rotation already won). This return value is the
	// only source of truth for whether the rotation happened — callers must not fall back
	// on an in-memory RefreshToken's IsRevoked() to decide anything after calling this.
	Rotate(ctx context.Context, oldTokenHash string, newToken *RefreshToken) (rotated bool, err error)

	RevokeByFamilyID(ctx context.Context, familyID string, now time.Time) error
	RevokeAllByUserID(ctx context.Context, userID string, now time.Time) error
	DeleteRevokedOrExpiredBefore(ctx context.Context, cutoff time.Time) error
}
