package user

import (
	"context"
	"time"

	"github.com/training-judge-center/backend/internal/domain/notification"
	domain "github.com/training-judge-center/backend/internal/domain/user"
)

type mockEmailSender struct {
	sendFn func(ctx context.Context, msg notification.EmailMessage) error
}

func (m *mockEmailSender) Send(ctx context.Context, msg notification.EmailMessage) error {
	if m.sendFn != nil {
		return m.sendFn(ctx, msg)
	}
	return nil
}

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

type mockUserRepository struct {
	saveFn            func(ctx context.Context, u *domain.User) error
	findByIDFn        func(ctx context.Context, id string) (*domain.User, error)
	findByEmailFn     func(ctx context.Context, email domain.Email) (*domain.User, error)
	findByNicknameFn  func(ctx context.Context, nickname domain.Nickname) (*domain.User, error)
	existsByEmailFn   func(ctx context.Context, email domain.Email) (bool, error)
	existsByNicknameFn func(ctx context.Context, nickname domain.Nickname) (bool, error)
	updateFn          func(ctx context.Context, u *domain.User) error
	findAllFn         func(ctx context.Context, filter domain.UserFilter) ([]*domain.User, int, error)
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
func (m *mockUserRepository) ExistsByEmail(ctx context.Context, email domain.Email) (bool, error) {
	if m.existsByEmailFn != nil {
		return m.existsByEmailFn(ctx, email)
	}
	return false, nil
}
func (m *mockUserRepository) ExistsByNickname(ctx context.Context, nickname domain.Nickname) (bool, error) {
	if m.existsByNicknameFn != nil {
		return m.existsByNicknameFn(ctx, nickname)
	}
	return false, nil
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

func newUserWithRole(id string, role domain.Role, status domain.Status) *domain.User {
	p, _ := domain.NewPassword("Secret1!")
	emailStr := id + "@example.com"
	nicknameStr := id

	u, err := domain.RestoreUser(
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
	if err != nil {
		panic("newUserWithRole: " + err.Error())
	}
	return u
}

type mockEmailChangeRepo struct {
	saveFn                func(ctx context.Context, req *domain.EmailChangeRequest) error
	findByIDFn            func(ctx context.Context, id string) (*domain.EmailChangeRequest, error)
	findByCodeAndUserIDFn func(ctx context.Context, code string, userID string) (*domain.EmailChangeRequest, error)
	invalidatePendingFn   func(ctx context.Context, userID string) error
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

func (m *mockEmailChangeRepo) InvalidatePendingByUserID(ctx context.Context, userID string) error {
	if m.invalidatePendingFn != nil {
		return m.invalidatePendingFn(ctx, userID)
	}
	return nil
}

func (m *mockEmailChangeRepo) Update(ctx context.Context, req *domain.EmailChangeRequest) error {
	if m.updateFn != nil {
		return m.updateFn(ctx, req)
	}
	return nil
}
