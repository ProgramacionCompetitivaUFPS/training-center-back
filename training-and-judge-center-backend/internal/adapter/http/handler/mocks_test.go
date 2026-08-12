package handler

import (
	"context"
	"time"

	appshared "github.com/training-judge-center/backend/internal/application/shared"
	appuser "github.com/training-judge-center/backend/internal/application/user"
	domainuser "github.com/training-judge-center/backend/internal/domain/user"
)

type mockUserRepo struct {
	findByEmailFn func(ctx context.Context, email domainuser.Email) (*domainuser.User, error)
	findByIDFn    func(ctx context.Context, id string) (*domainuser.User, error)
}

func (m *mockUserRepo) Save(_ context.Context, _ *domainuser.User) error   { return nil }
func (m *mockUserRepo) Update(_ context.Context, _ *domainuser.User) error { return nil }
func (m *mockUserRepo) FindByEmail(ctx context.Context, email domainuser.Email) (*domainuser.User, error) {
	if m.findByEmailFn != nil {
		return m.findByEmailFn(ctx, email)
	}
	return nil, nil
}
func (m *mockUserRepo) FindByID(ctx context.Context, id string) (*domainuser.User, error) {
	if m.findByIDFn != nil {
		return m.findByIDFn(ctx, id)
	}
	return nil, nil
}
func (m *mockUserRepo) FindByNickname(_ context.Context, _ domainuser.Nickname) (*domainuser.User, error) {
	return nil, nil
}
func (m *mockUserRepo) FindAll(_ context.Context, _ domainuser.UserFilter) ([]*domainuser.User, int, error) {
	return nil, 0, nil
}

type mockRefreshTokenRepo struct {
	saveFn                 func(ctx context.Context, token *domainuser.RefreshToken) error
	findByTokenHashFn      func(ctx context.Context, tokenHash string) (*domainuser.RefreshToken, error)
	findActiveByFamilyIDFn func(ctx context.Context, familyID string) (*domainuser.RefreshToken, error)
	rotateFn               func(ctx context.Context, oldTokenHash string, newToken *domainuser.RefreshToken) (bool, error)
	revokeByFamilyIDFn     func(ctx context.Context, familyID string, now time.Time) error
}

func (m *mockRefreshTokenRepo) Save(ctx context.Context, token *domainuser.RefreshToken) error {
	if m.saveFn != nil {
		return m.saveFn(ctx, token)
	}
	return nil
}
func (m *mockRefreshTokenRepo) FindByTokenHash(ctx context.Context, tokenHash string) (*domainuser.RefreshToken, error) {
	if m.findByTokenHashFn != nil {
		return m.findByTokenHashFn(ctx, tokenHash)
	}
	return nil, nil
}
func (m *mockRefreshTokenRepo) FindActiveByFamilyID(ctx context.Context, familyID string) (*domainuser.RefreshToken, error) {
	if m.findActiveByFamilyIDFn != nil {
		return m.findActiveByFamilyIDFn(ctx, familyID)
	}
	return nil, nil
}
func (m *mockRefreshTokenRepo) Rotate(ctx context.Context, oldTokenHash string, newToken *domainuser.RefreshToken) (bool, error) {
	if m.rotateFn != nil {
		return m.rotateFn(ctx, oldTokenHash, newToken)
	}
	return true, nil
}
func (m *mockRefreshTokenRepo) RevokeByFamilyID(ctx context.Context, familyID string, now time.Time) error {
	if m.revokeByFamilyIDFn != nil {
		return m.revokeByFamilyIDFn(ctx, familyID, now)
	}
	return nil
}
func (m *mockRefreshTokenRepo) RevokeAllByUserID(_ context.Context, _ string, _ time.Time) error {
	return nil
}
func (m *mockRefreshTokenRepo) DeleteRevokedOrExpiredBefore(_ context.Context, _ time.Time) error {
	return nil
}

type mockTokenService struct {
	generateTokenFn func(ctx context.Context, u *domainuser.User, sessionID string) (string, error)
	validateTokenFn func(token string) (*domainuser.TokenClaims, error)
}

func (m *mockTokenService) GenerateToken(ctx context.Context, u *domainuser.User, sessionID string) (string, error) {
	if m.generateTokenFn != nil {
		return m.generateTokenFn(ctx, u, sessionID)
	}
	return "access-token", nil
}
func (m *mockTokenService) ValidateToken(token string) (*domainuser.TokenClaims, error) {
	if m.validateTokenFn != nil {
		return m.validateTokenFn(token)
	}
	return nil, nil
}

type mockRateLimiter struct {
	allowFn func(ctx context.Context, key string, maxAttempts int, window time.Duration) (bool, error)
}

func (m *mockRateLimiter) Allow(ctx context.Context, key string, maxAttempts int, window time.Duration) (bool, error) {
	if m.allowFn != nil {
		return m.allowFn(ctx, key, maxAttempts, window)
	}
	return true, nil
}
func (m *mockRateLimiter) Reset(_ context.Context, _ string) error { return nil }

var _ appshared.RateLimiter = (*mockRateLimiter)(nil)

type mockRefreshTokenCodec struct {
	wrapFn   func(ctx context.Context, secret, userID string) (string, error)
	unwrapFn func(wrapped string) (secret, userID string, err error)
}

func (m *mockRefreshTokenCodec) Wrap(ctx context.Context, secret, userID string) (string, error) {
	if m.wrapFn != nil {
		return m.wrapFn(ctx, secret, userID)
	}
	return secret, nil
}

func (m *mockRefreshTokenCodec) Unwrap(wrapped string) (string, string, error) {
	if m.unwrapFn != nil {
		return m.unwrapFn(wrapped)
	}
	return wrapped, "user-id", nil
}

type mockSessionInvalidator struct {
	invalidateAllUserSessionsFn func(ctx context.Context, userID string, timestamp time.Time) error
	isAllUserSessionRevokedFn   func(ctx context.Context, userID string, tokenIssuedAt time.Time) (bool, error)
	invalidateSessionFn         func(ctx context.Context, sessionID string, timestamp time.Time) error
	isSessionInvalidatedFn      func(ctx context.Context, sessionID string) (bool, error)
}

func (m *mockSessionInvalidator) InvalidateAllUserSessions(ctx context.Context, userID string, timestamp time.Time) error {
	if m.invalidateAllUserSessionsFn != nil {
		return m.invalidateAllUserSessionsFn(ctx, userID, timestamp)
	}
	return nil
}
func (m *mockSessionInvalidator) IsAllUserSessionRevoked(ctx context.Context, userID string, tokenIssuedAt time.Time) (bool, error) {
	if m.isAllUserSessionRevokedFn != nil {
		return m.isAllUserSessionRevokedFn(ctx, userID, tokenIssuedAt)
	}
	return false, nil
}
func (m *mockSessionInvalidator) InvalidateSession(ctx context.Context, sessionID string, timestamp time.Time) error {
	if m.invalidateSessionFn != nil {
		return m.invalidateSessionFn(ctx, sessionID, timestamp)
	}
	return nil
}
func (m *mockSessionInvalidator) IsSessionInvalidated(ctx context.Context, sessionID string) (bool, error) {
	if m.isSessionInvalidatedFn != nil {
		return m.isSessionInvalidatedFn(ctx, sessionID)
	}
	return false, nil
}

type mockRotationCache struct {
	getFn  func(ctx context.Context, oldTokenHash string) (*appuser.RefreshOutput, error)
	saveFn func(ctx context.Context, oldTokenHash string, output appuser.RefreshOutput, ttl time.Duration) error
}

func (m *mockRotationCache) Save(ctx context.Context, oldTokenHash string, output appuser.RefreshOutput, ttl time.Duration) error {
	if m.saveFn != nil {
		return m.saveFn(ctx, oldTokenHash, output, ttl)
	}
	return nil
}
func (m *mockRotationCache) Get(ctx context.Context, oldTokenHash string) (*appuser.RefreshOutput, error) {
	if m.getFn != nil {
		return m.getFn(ctx, oldTokenHash)
	}
	return nil, nil
}
