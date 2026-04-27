package group

import "context"

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
	ID   string
	Role string
}

type NicknameResolver interface {
	ResolveByNickname(ctx context.Context, nickname string) (*UserInfo, error)
}

type TransactionManager interface {
	WithTx(ctx context.Context, fn func(ctx context.Context) error) error
}
