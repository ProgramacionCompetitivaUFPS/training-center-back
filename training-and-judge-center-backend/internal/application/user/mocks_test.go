package user

import (
	"context"
	"time"

	appshared "github.com/training-judge-center/backend/internal/application/shared"
	"github.com/training-judge-center/backend/internal/domain/shared"
	domain "github.com/training-judge-center/backend/internal/domain/user"
)

// ── mockRateLimiter ───────────────────────────────────────────────────────────

type mockRateLimiter struct {
	allowFn func(ctx context.Context, key string, maxAttempts int, window time.Duration) (bool, error)
	resetFn func(ctx context.Context, key string) error
}

func (m *mockRateLimiter) Allow(ctx context.Context, key string, maxAttempts int, window time.Duration) (bool, error) {
	if m.allowFn != nil {
		return m.allowFn(ctx, key, maxAttempts, window)
	}
	return true, nil
}

func (m *mockRateLimiter) Reset(ctx context.Context, key string) error {
	if m.resetFn != nil {
		return m.resetFn(ctx, key)
	}
	return nil
}

// ── mockEmailSender ───────────────────────────────────────────────────────────

type mockEmailSender struct {
	sendFn func(ctx context.Context, msg appshared.EmailMessage) error
}

func (m *mockEmailSender) Send(ctx context.Context, msg appshared.EmailMessage) error {
	if m.sendFn != nil {
		return m.sendFn(ctx, msg)
	}
	return nil
}

// ── mockSessionInvalidator ────────────────────────────────────────────────────

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

// ── mockUserRepository ───────────────────────────────────────────────────────

type mockUserRepository struct {
	saveFn           func(ctx context.Context, u *domain.User) error
	findByIDFn       func(ctx context.Context, id string) (*domain.User, error)
	findByEmailFn    func(ctx context.Context, email domain.Email) (*domain.User, error)
	findByNicknameFn func(ctx context.Context, nickname domain.Nickname) (*domain.User, error)
	updateFn         func(ctx context.Context, u *domain.User) error
	findAllFn        func(ctx context.Context, filter domain.UserFilter) ([]*domain.User, int, error)
}

func (m *mockUserRepository) Save(ctx context.Context, u *domain.User) error {
	if m.saveFn != nil {
		return m.saveFn(ctx, u)
	}
	return nil
}
func (m *mockUserRepository) FindByID(ctx context.Context, id string) (*domain.User, error) {
	if m.findByIDFn != nil {
		return m.findByIDFn(ctx, id)
	}
	return nil, nil
}
func (m *mockUserRepository) FindByEmail(ctx context.Context, email domain.Email) (*domain.User, error) {
	if m.findByEmailFn != nil {
		return m.findByEmailFn(ctx, email)
	}
	return nil, nil
}
func (m *mockUserRepository) FindByNickname(ctx context.Context, nickname domain.Nickname) (*domain.User, error) {
	if m.findByNicknameFn != nil {
		return m.findByNicknameFn(ctx, nickname)
	}
	return nil, nil
}
func (m *mockUserRepository) Update(ctx context.Context, u *domain.User) error {
	if m.updateFn != nil {
		return m.updateFn(ctx, u)
	}
	return nil
}
func (m *mockUserRepository) FindAll(ctx context.Context, filter domain.UserFilter) ([]*domain.User, int, error) {
	if m.findAllFn != nil {
		return m.findAllFn(ctx, filter)
	}
	return nil, 0, nil
}

func newNoConflictRepo() *mockUserRepository {
	return &mockUserRepository{}
}

func newUserWithRole(id string, role shared.Role, status domain.Status) *domain.User {
	p, _ := domain.NewPassword("Secret1!")
	emailStr := id + "@example.com"
	nicknameStr := id

	return domain.RestoreUser(
		id,
		&emailStr,
		p.Hash(),
		"User "+id,
		nicknameStr,
		"",
		"",
		"",
		role.String(),
		status.String(),
		time.Now(),
		nil,
		nil,
	)
}

// ── mockEmailChangeRepo ───────────────────────────────────────────────────────

type mockEmailChangeRepo struct {
	saveFn                func(ctx context.Context, req *domain.EmailChangeRequest) error
	findByIDFn            func(ctx context.Context, id string) (*domain.EmailChangeRequest, error)
	findByCodeAndUserIDFn func(ctx context.Context, code string, userID string) (*domain.EmailChangeRequest, error)
	invalidatePendingFn   func(ctx context.Context, userID string, now time.Time) error
	updateFn              func(ctx context.Context, req *domain.EmailChangeRequest) error
}

func (m *mockEmailChangeRepo) Save(ctx context.Context, req *domain.EmailChangeRequest) error {
	if m.saveFn != nil {
		return m.saveFn(ctx, req)
	}
	return nil
}

func (m *mockEmailChangeRepo) FindByID(ctx context.Context, id string) (*domain.EmailChangeRequest, error) {
	if m.findByIDFn != nil {
		return m.findByIDFn(ctx, id)
	}
	return nil, nil
}

func (m *mockEmailChangeRepo) FindByCodeAndUserID(ctx context.Context, code string, userID string) (*domain.EmailChangeRequest, error) {
	if m.findByCodeAndUserIDFn != nil {
		return m.findByCodeAndUserIDFn(ctx, code, userID)
	}
	return nil, nil
}

func (m *mockEmailChangeRepo) InvalidatePendingByUserID(ctx context.Context, userID string, now time.Time) error {
	if m.invalidatePendingFn != nil {
		return m.invalidatePendingFn(ctx, userID, now)
	}
	return nil
}

func (m *mockEmailChangeRepo) Update(ctx context.Context, req *domain.EmailChangeRequest) error {
	if m.updateFn != nil {
		return m.updateFn(ctx, req)
	}
	return nil
}

// ── mockTransactionManager ───────────────────────────────────────────────────

type mockTransactionManager struct {
	withTxFn func(ctx context.Context, fn func(txCtx context.Context) error) error
}

func (m *mockTransactionManager) WithTx(ctx context.Context, fn func(txCtx context.Context) error) error {
	if m.withTxFn != nil {
		return m.withTxFn(ctx, fn)
	}
	return fn(ctx)
}

// ── mockDeactivationRepo ──────────────────────────────────────────────────────

type mockDeactivationRepo struct {
	saveFn                      func(ctx context.Context, req *domain.DeactivationRequest) error
	findByIDFn                  func(ctx context.Context, id string) (*domain.DeactivationRequest, error)
	updateFn                    func(ctx context.Context, req *domain.DeactivationRequest) error
	findPendingByUserIDFn       func(ctx context.Context, userID string) (*domain.DeactivationRequest, error)
	invalidatePendingByUserIDFn func(ctx context.Context, userID string, now time.Time) error
}

func (m *mockDeactivationRepo) Save(ctx context.Context, req *domain.DeactivationRequest) error {
	if m.saveFn != nil {
		return m.saveFn(ctx, req)
	}
	return nil
}
func (m *mockDeactivationRepo) FindByID(ctx context.Context, id string) (*domain.DeactivationRequest, error) {
	if m.findByIDFn != nil {
		return m.findByIDFn(ctx, id)
	}
	return nil, nil
}
func (m *mockDeactivationRepo) Update(ctx context.Context, req *domain.DeactivationRequest) error {
	if m.updateFn != nil {
		return m.updateFn(ctx, req)
	}
	return nil
}
func (m *mockDeactivationRepo) FindPendingByUserID(ctx context.Context, userID string) (*domain.DeactivationRequest, error) {
	if m.findPendingByUserIDFn != nil {
		return m.findPendingByUserIDFn(ctx, userID)
	}
	return nil, nil
}
func (m *mockDeactivationRepo) InvalidatePendingByUserID(ctx context.Context, userID string, now time.Time) error {
	if m.invalidatePendingByUserIDFn != nil {
		return m.invalidatePendingByUserIDFn(ctx, userID, now)
	}
	return nil
}

// ── mockPasswordRecoveryRepo ──────────────────────────────────────────────────

type mockPasswordRecoveryRepo struct {
	saveFn                      func(ctx context.Context, req *domain.PasswordRecoveryRequest) error
	findByIDFn                  func(ctx context.Context, id string) (*domain.PasswordRecoveryRequest, error)
	updateFn                    func(ctx context.Context, req *domain.PasswordRecoveryRequest) error
	findPendingByUserIDFn       func(ctx context.Context, userID string) (*domain.PasswordRecoveryRequest, error)
	invalidatePendingByUserIDFn func(ctx context.Context, userID string, now time.Time) error
}

func (m *mockPasswordRecoveryRepo) Save(ctx context.Context, req *domain.PasswordRecoveryRequest) error {
	if m.saveFn != nil {
		return m.saveFn(ctx, req)
	}
	return nil
}
func (m *mockPasswordRecoveryRepo) FindByID(ctx context.Context, id string) (*domain.PasswordRecoveryRequest, error) {
	if m.findByIDFn != nil {
		return m.findByIDFn(ctx, id)
	}
	return nil, nil
}
func (m *mockPasswordRecoveryRepo) Update(ctx context.Context, req *domain.PasswordRecoveryRequest) error {
	if m.updateFn != nil {
		return m.updateFn(ctx, req)
	}
	return nil
}
func (m *mockPasswordRecoveryRepo) FindPendingByUserID(ctx context.Context, userID string) (*domain.PasswordRecoveryRequest, error) {
	if m.findPendingByUserIDFn != nil {
		return m.findPendingByUserIDFn(ctx, userID)
	}
	return nil, nil
}
func (m *mockPasswordRecoveryRepo) InvalidatePendingByUserID(ctx context.Context, userID string, now time.Time) error {
	if m.invalidatePendingByUserIDFn != nil {
		return m.invalidatePendingByUserIDFn(ctx, userID, now)
	}
	return nil
}

// ── mockRefreshTokenRepository ───────────────────────────────────────────────

type mockRefreshTokenRepository struct {
	saveFn                         func(ctx context.Context, token *domain.RefreshToken) error
	findByTokenHashFn              func(ctx context.Context, tokenHash string) (*domain.RefreshToken, error)
	findActiveByFamilyIDFn         func(ctx context.Context, familyID string) (*domain.RefreshToken, error)
	rotateFn                       func(ctx context.Context, oldTokenHash string, newToken *domain.RefreshToken) (bool, error)
	revokeByFamilyIDFn             func(ctx context.Context, familyID string, now time.Time) error
	revokeAllByUserIDFn            func(ctx context.Context, userID string, now time.Time) error
	deleteRevokedOrExpiredBeforeFn func(ctx context.Context, cutoff time.Time) error
}

func (m *mockRefreshTokenRepository) Save(ctx context.Context, token *domain.RefreshToken) error {
	if m.saveFn != nil {
		return m.saveFn(ctx, token)
	}
	return nil
}
func (m *mockRefreshTokenRepository) FindByTokenHash(ctx context.Context, tokenHash string) (*domain.RefreshToken, error) {
	if m.findByTokenHashFn != nil {
		return m.findByTokenHashFn(ctx, tokenHash)
	}
	return nil, nil
}
func (m *mockRefreshTokenRepository) FindActiveByFamilyID(ctx context.Context, familyID string) (*domain.RefreshToken, error) {
	if m.findActiveByFamilyIDFn != nil {
		return m.findActiveByFamilyIDFn(ctx, familyID)
	}
	return nil, nil
}
func (m *mockRefreshTokenRepository) Rotate(ctx context.Context, oldTokenHash string, newToken *domain.RefreshToken) (bool, error) {
	if m.rotateFn != nil {
		return m.rotateFn(ctx, oldTokenHash, newToken)
	}
	return true, nil
}
func (m *mockRefreshTokenRepository) RevokeByFamilyID(ctx context.Context, familyID string, now time.Time) error {
	if m.revokeByFamilyIDFn != nil {
		return m.revokeByFamilyIDFn(ctx, familyID, now)
	}
	return nil
}
func (m *mockRefreshTokenRepository) RevokeAllByUserID(ctx context.Context, userID string, now time.Time) error {
	if m.revokeAllByUserIDFn != nil {
		return m.revokeAllByUserIDFn(ctx, userID, now)
	}
	return nil
}
func (m *mockRefreshTokenRepository) DeleteRevokedOrExpiredBefore(ctx context.Context, cutoff time.Time) error {
	if m.deleteRevokedOrExpiredBeforeFn != nil {
		return m.deleteRevokedOrExpiredBeforeFn(ctx, cutoff)
	}
	return nil
}

// ── mockRotationCache ─────────────────────────────────────────────────────────

type mockRotationCache struct {
	saveFn func(ctx context.Context, oldTokenHash string, output RefreshOutput, ttl time.Duration) error
	getFn  func(ctx context.Context, oldTokenHash string) (*RefreshOutput, error)
}

func (m *mockRotationCache) Save(ctx context.Context, oldTokenHash string, output RefreshOutput, ttl time.Duration) error {
	if m.saveFn != nil {
		return m.saveFn(ctx, oldTokenHash, output, ttl)
	}
	return nil
}
func (m *mockRotationCache) Get(ctx context.Context, oldTokenHash string) (*RefreshOutput, error) {
	if m.getFn != nil {
		return m.getFn(ctx, oldTokenHash)
	}
	return nil, nil
}

// ── mockRefreshTokenCodec ─────────────────────────────────────────────────────

// Defaults to a pass-through Wrap and an Unwrap that reports testRefreshUserID, so existing
// fixtures built around raw secret strings keep working without needing a real JWT envelope.
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
	return wrapped, testRefreshUserID, nil
}

// ── mockOAuthIdentityRepository ──────────────────────────────────────────────

type mockOAuthIdentityRepository struct {
	saveFn           func(ctx context.Context, identity *domain.OAuthIdentity) error
	findByProviderFn func(ctx context.Context, provider domain.OAuthProvider, providerUserID string) (*domain.OAuthIdentity, error)
	findByUserIDFn   func(ctx context.Context, userID string, provider domain.OAuthProvider) (*domain.OAuthIdentity, error)
	deleteByUserIDFn func(ctx context.Context, userID string, provider domain.OAuthProvider) (bool, error)
}

func (m *mockOAuthIdentityRepository) Save(ctx context.Context, identity *domain.OAuthIdentity) error {
	if m.saveFn != nil {
		return m.saveFn(ctx, identity)
	}
	return nil
}

func (m *mockOAuthIdentityRepository) FindByProvider(ctx context.Context, provider domain.OAuthProvider, providerUserID string) (*domain.OAuthIdentity, error) {
	if m.findByProviderFn != nil {
		return m.findByProviderFn(ctx, provider, providerUserID)
	}
	return nil, nil
}

func (m *mockOAuthIdentityRepository) FindByUserID(ctx context.Context, userID string, provider domain.OAuthProvider) (*domain.OAuthIdentity, error) {
	if m.findByUserIDFn != nil {
		return m.findByUserIDFn(ctx, userID, provider)
	}
	return nil, nil
}

func (m *mockOAuthIdentityRepository) DeleteByUserID(ctx context.Context, userID string, provider domain.OAuthProvider) (bool, error) {
	if m.deleteByUserIDFn != nil {
		return m.deleteByUserIDFn(ctx, userID, provider)
	}
	return true, nil
}

// ── mockGoogleIDTokenVerifier ────────────────────────────────────────────────

type mockGoogleIDTokenVerifier struct {
	verifyFn func(ctx context.Context, idToken string) (*GoogleClaims, error)
}

func (m *mockGoogleIDTokenVerifier) Verify(ctx context.Context, idToken string) (*GoogleClaims, error) {
	if m.verifyFn != nil {
		return m.verifyFn(ctx, idToken)
	}
	return &GoogleClaims{Sub: "google-sub", Email: "google@example.com", EmailVerified: true, Name: "Google User"}, nil
}

// ── helpers ───────────────────────────────────────────────────────────────────

func strPtr(s string) *string { return &s }
