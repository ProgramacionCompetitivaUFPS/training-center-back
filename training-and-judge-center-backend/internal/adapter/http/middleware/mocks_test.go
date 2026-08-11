package middleware

import (
	"context"
	"time"

	"github.com/training-judge-center/backend/internal/domain/user"
)

type mockTokenService struct {
	validateTokenFn func(tokenString string) (*user.TokenClaims, error)
}

func (m *mockTokenService) GenerateToken(_ context.Context, _ *user.User, _ string) (string, error) {
	return "", nil
}

func (m *mockTokenService) ValidateToken(tokenString string) (*user.TokenClaims, error) {
	if m.validateTokenFn != nil {
		return m.validateTokenFn(tokenString)
	}
	return nil, nil
}

type mockSessionInvalidator struct {
	isAllUserSessionRevokedFn func(ctx context.Context, userID string, tokenIssuedAt time.Time) (bool, error)
	isSessionInvalidatedFn    func(ctx context.Context, sessionID string, tokenIssuedAt time.Time) (bool, error)
}

func (m *mockSessionInvalidator) InvalidateAllUserSessions(_ context.Context, _ string, _ time.Time) error {
	return nil
}

func (m *mockSessionInvalidator) IsAllUserSessionRevoked(ctx context.Context, userID string, tokenIssuedAt time.Time) (bool, error) {
	if m.isAllUserSessionRevokedFn != nil {
		return m.isAllUserSessionRevokedFn(ctx, userID, tokenIssuedAt)
	}
	return false, nil
}

func (m *mockSessionInvalidator) InvalidateSession(_ context.Context, _ string, _ time.Time) error {
	return nil
}

func (m *mockSessionInvalidator) IsSessionInvalidated(ctx context.Context, sessionID string, tokenIssuedAt time.Time) (bool, error) {
	if m.isSessionInvalidatedFn != nil {
		return m.isSessionInvalidatedFn(ctx, sessionID, tokenIssuedAt)
	}
	return false, nil
}
