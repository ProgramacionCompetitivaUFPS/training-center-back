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
	isSessionRevokedFn          func(ctx context.Context, userID string, tokenIssuedAt time.Time) (bool, error)
}

func (m *mockSessionInvalidator) InvalidateAllUserSessions(ctx context.Context, userID string, timestamp time.Time) error {
	if m.invalidateAllUserSessionsFn != nil {
		return m.invalidateAllUserSessionsFn(ctx, userID, timestamp)
	}
	return nil
}

func (m *mockSessionInvalidator) IsSessionRevoked(ctx context.Context, userID string, tokenIssuedAt time.Time) (bool, error) {
	if m.isSessionRevokedFn != nil {
		return m.isSessionRevokedFn(ctx, userID, tokenIssuedAt)
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

// ── helpers ───────────────────────────────────────────────────────────────────

func strPtr(s string) *string { return &s }
