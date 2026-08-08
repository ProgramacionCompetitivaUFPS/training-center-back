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
	// rotated is the only source of truth for whether it happened — false means reuse or
	// a concurrent rotation already won, not success. Any failure other than a genuine
	// zero-rows-matched must return a non-nil err — never collapse an unrelated failure
	// into rotated=false, since callers treat false as grounds to revoke the whole family.
	Rotate(ctx context.Context, oldTokenHash string, newToken *RefreshToken) (rotated bool, err error)

	RevokeByFamilyID(ctx context.Context, familyID string, now time.Time) error
	RevokeAllByUserID(ctx context.Context, userID string, now time.Time) error
	DeleteRevokedOrExpiredBefore(ctx context.Context, cutoff time.Time) error
}
