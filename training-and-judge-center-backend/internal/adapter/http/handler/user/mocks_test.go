package user

import (
	"context"
	"time"

	"github.com/training-judge-center/backend/internal/application/shared"
	domainuser "github.com/training-judge-center/backend/internal/domain/user"
)

// mockHandlerEmailChangeRepo implements domainuser.EmailChangeRepository for handler tests.
type mockHandlerEmailChangeRepo struct {
	findByCodeAndUserIDFn func(ctx context.Context, code string, userID string) (*domainuser.EmailChangeRequest, error)
}

func (m *mockHandlerEmailChangeRepo) Save(_ context.Context, _ *domainuser.EmailChangeRequest) error {
	return nil
}
func (m *mockHandlerEmailChangeRepo) FindByID(_ context.Context, _ string) (*domainuser.EmailChangeRequest, error) {
	return nil, nil
}
func (m *mockHandlerEmailChangeRepo) FindByCodeAndUserID(ctx context.Context, code string, userID string) (*domainuser.EmailChangeRequest, error) {
	if m.findByCodeAndUserIDFn != nil {
		return m.findByCodeAndUserIDFn(ctx, code, userID)
	}
	return nil, nil
}
func (m *mockHandlerEmailChangeRepo) InvalidatePendingByUserID(_ context.Context, _ string, _ time.Time) error {
	return nil
}
func (m *mockHandlerEmailChangeRepo) Update(_ context.Context, _ *domainuser.EmailChangeRequest) error {
	return nil
}

// mockHandlerPasswordRecoveryRepo implements domainuser.PasswordRecoveryRepository for handler tests.
type mockHandlerPasswordRecoveryRepo struct {
	findPendingByUserIDFn func(ctx context.Context, userID string) (*domainuser.PasswordRecoveryRequest, error)
	updateFn              func(ctx context.Context, req *domainuser.PasswordRecoveryRequest) error
}

func (m *mockHandlerPasswordRecoveryRepo) Save(_ context.Context, _ *domainuser.PasswordRecoveryRequest) error {
	return nil
}
func (m *mockHandlerPasswordRecoveryRepo) FindByID(_ context.Context, _ string) (*domainuser.PasswordRecoveryRequest, error) {
	return nil, nil
}
func (m *mockHandlerPasswordRecoveryRepo) FindPendingByUserID(ctx context.Context, userID string) (*domainuser.PasswordRecoveryRequest, error) {
	if m.findPendingByUserIDFn != nil {
		return m.findPendingByUserIDFn(ctx, userID)
	}
	return nil, nil
}
func (m *mockHandlerPasswordRecoveryRepo) Update(ctx context.Context, req *domainuser.PasswordRecoveryRequest) error {
	if m.updateFn != nil {
		return m.updateFn(ctx, req)
	}
	return nil
}
func (m *mockHandlerPasswordRecoveryRepo) InvalidatePendingByUserID(_ context.Context, _ string, _ time.Time) error {
	return nil
}

// mockHandlerRateLimiter implements ratelimit.RateLimiter for handler tests.
// By default it allows all requests.
type mockHandlerRateLimiter struct {
	allowFn func(ctx context.Context, key string, maxAttempts int, window time.Duration) (bool, error)
}

func (m *mockHandlerRateLimiter) Allow(ctx context.Context, key string, maxAttempts int, window time.Duration) (bool, error) {
	if m.allowFn != nil {
		return m.allowFn(ctx, key, maxAttempts, window)
	}
	return true, nil
}

func (m *mockHandlerRateLimiter) Reset(_ context.Context, _ string) error { return nil }

type mockTokenService struct {
	validateFn func(token string) (*domainuser.TokenClaims, error)
}

func (m *mockTokenService) GenerateToken(_ *domainuser.User) (string, error) { return "", nil }
func (m *mockTokenService) ValidateToken(token string) (*domainuser.TokenClaims, error) {
	return m.validateFn(token)
}

type mockHandlerUserRepo struct {
	findByIDFn    func(ctx context.Context, id string) (*domainuser.User, error)
	findByEmailFn func(ctx context.Context, email domainuser.Email) (*domainuser.User, error)
	updateFn      func(ctx context.Context, u *domainuser.User) error
}

func (m *mockHandlerUserRepo) Save(_ context.Context, _ *domainuser.User) error { return nil }
func (m *mockHandlerUserRepo) Update(ctx context.Context, u *domainuser.User) error {
	if m.updateFn != nil {
		return m.updateFn(ctx, u)
	}
	return nil
}
func (m *mockHandlerUserRepo) FindByEmail(ctx context.Context, email domainuser.Email) (*domainuser.User, error) {
	if m.findByEmailFn != nil {
		return m.findByEmailFn(ctx, email)
	}
	return nil, nil
}
func (m *mockHandlerUserRepo) FindByID(ctx context.Context, id string) (*domainuser.User, error) {
	if m.findByIDFn != nil {
		return m.findByIDFn(ctx, id)
	}
	return nil, nil
}
func (m *mockHandlerUserRepo) FindByNickname(_ context.Context, _ domainuser.Nickname) (*domainuser.User, error) {
	return nil, nil
}
func (m *mockHandlerUserRepo) FindAll(_ context.Context, _ domainuser.UserFilter) ([]*domainuser.User, int, error) {
	return nil, 0, nil
}

type mockHandlerDeactRepo struct {
	findPendingByUserIDFn func(ctx context.Context, userID string) (*domainuser.DeactivationRequest, error)
	updateFn              func(ctx context.Context, req *domainuser.DeactivationRequest) error
}

func (m *mockHandlerDeactRepo) Save(_ context.Context, _ *domainuser.DeactivationRequest) error {
	return nil
}
func (m *mockHandlerDeactRepo) FindPendingByUserID(ctx context.Context, userID string) (*domainuser.DeactivationRequest, error) {
	if m.findPendingByUserIDFn != nil {
		return m.findPendingByUserIDFn(ctx, userID)
	}
	return nil, nil
}
func (m *mockHandlerDeactRepo) FindByID(_ context.Context, _ string) (*domainuser.DeactivationRequest, error) {
	return nil, nil
}
func (m *mockHandlerDeactRepo) Update(ctx context.Context, req *domainuser.DeactivationRequest) error {
	if m.updateFn != nil {
		return m.updateFn(ctx, req)
	}
	return nil
}
func (m *mockHandlerDeactRepo) InvalidatePendingByUserID(_ context.Context, _ string, _ time.Time) error {
	return nil
}

type mockHandlerDeactAuditRepo struct{}

func (m *mockHandlerDeactAuditRepo) Save(_ context.Context, _ *domainuser.DeactivationAuditLog) error {
	return nil
}

type mockHandlerEmailSender struct{}

func (m *mockHandlerEmailSender) Send(_ context.Context, _ shared.EmailMessage) error {
	return nil
}

type mockHandlerSessionInvalidator struct {
	invalidateAllUserSessionsFn func(ctx context.Context, userID string, timestamp time.Time) error
}

func (m *mockHandlerSessionInvalidator) InvalidateAllUserSessions(ctx context.Context, userID string, timestamp time.Time) error {
	if m.invalidateAllUserSessionsFn != nil {
		return m.invalidateAllUserSessionsFn(ctx, userID, timestamp)
	}
	return nil
}
func (m *mockHandlerSessionInvalidator) IsSessionRevoked(_ context.Context, _ string, _ time.Time) (bool, error) {
	return false, nil
}

type mockHandlerTxManager struct{}

func (m *mockHandlerTxManager) WithTx(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}
