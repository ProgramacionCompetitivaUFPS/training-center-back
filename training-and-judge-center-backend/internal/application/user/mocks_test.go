package user

import (
	"context"
	"time"

	domain "github.com/training-judge-center/backend/internal/domain/user"
)

type mockEmailSender struct {
	sendFn func(ctx context.Context, to, subject, body string) error
}

func (m *mockEmailSender) Send(ctx context.Context, to, subject, body string) error {
	if m.sendFn != nil {
		return m.sendFn(ctx, to, subject, body)
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
	email, _ := domain.NewEmail(id + "@example.com")
	nickname, _ := domain.NewNickname(id)
	u := &domain.User{
		ID:        id,
		Email:     &email,
		Nickname:  nickname,
		Name:      "User " + id,
		Role:      role,
		Status:    status,
		CreatedAt: time.Now(),
	}
	p, _ := domain.NewPassword("Secret1!")
	u.Password = p
	return u
}
