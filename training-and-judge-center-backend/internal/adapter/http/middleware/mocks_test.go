package middleware

import (
	"context"
	"time"

	"github.com/training-judge-center/backend/internal/domain/user"
)

type mockTokenService struct {
	validateTokenFn func(tokenString string) (*user.TokenClaims, error)
}

func (m *mockTokenService) GenerateToken(_ context.Context, _ *user.User) (string, error) { return "", nil }

func (m *mockTokenService) ValidateToken(tokenString string) (*user.TokenClaims, error) {
	if m.validateTokenFn != nil {
		return m.validateTokenFn(tokenString)
	}
	return nil, nil
}

type mockSessionInvalidator struct {
	isSessionRevokedFn func(ctx context.Context, userID string, tokenIssuedAt time.Time) (bool, error)
}

func (m *mockSessionInvalidator) InvalidateAllUserSessions(_ context.Context, _ string, _ time.Time) error {
	return nil
}

func (m *mockSessionInvalidator) IsSessionRevoked(ctx context.Context, userID string, tokenIssuedAt time.Time) (bool, error) {
	if m.isSessionRevokedFn != nil {
		return m.isSessionRevokedFn(ctx, userID, tokenIssuedAt)
	}
	return false, nil
}
