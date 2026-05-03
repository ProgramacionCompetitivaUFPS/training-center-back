package group

import (
	"context"

	"github.com/training-judge-center/backend/internal/domain/shared"
)

type UserDisplay struct {
	Nickname string
	Name     string
}

type UserProvider interface {
	GetDisplays(ctx context.Context, userIDs []string) (map[string]*UserDisplay, error)
}

type PreferencesReader interface {
	HideGlobalGroup(ctx context.Context, userID string) (bool, error)
}

type UserInfo struct {
	ID   shared.UserID
	Role string
}

type NicknameResolver interface {
	ResolveByNickname(ctx context.Context, nickname string) (*UserInfo, error)
}

type TransactionManager interface {
	WithTx(ctx context.Context, fn func(ctx context.Context) error) error
}
