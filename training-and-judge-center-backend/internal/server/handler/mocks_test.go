package handler

import (
	"context"
	"time"

	"github.com/training-judge-center/backend/internal/domain/notification"
	domainuser "github.com/training-judge-center/backend/internal/domain/user"
)

type mockTokenService struct {
	validateFn func(token string) (*domainuser.TokenClaims, error)
}

func (m *mockTokenService) GenerateToken(_ *domainuser.User) (string, error) { return "", nil }
func (m *mockTokenService) ValidateToken(token string) (*domainuser.TokenClaims, error) {
	return m.validateFn(token)
}

type mockHandlerUserRepo struct {
	findByIDFn func(ctx context.Context, id string) (*domainuser.User, error)
	updateFn   func(ctx context.Context, u *domainuser.User) error
}

func (m *mockHandlerUserRepo) Save(_ context.Context, _ *domainuser.User) error { return nil }
func (m *mockHandlerUserRepo) Update(ctx context.Context, u *domainuser.User) error {
	if m.updateFn != nil {
		return m.updateFn(ctx, u)
	}
	return nil
}
func (m *mockHandlerUserRepo) FindByEmail(_ context.Context, _ domainuser.Email) (*domainuser.User, error) {
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

func (m *mockHandlerEmailSender) Send(_ context.Context, _ notification.EmailMessage) error {
	return nil
}

type mockHandlerSessionInvalidator struct{}

func (m *mockHandlerSessionInvalidator) InvalidateAllUserSessions(_ context.Context, _ string, _ time.Time) error {
	return nil
}
func (m *mockHandlerSessionInvalidator) IsSessionRevoked(_ context.Context, _ string, _ time.Time) (bool, error) {
	return false, nil
}

type mockHandlerTxManager struct{}

func (m *mockHandlerTxManager) WithTx(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}
