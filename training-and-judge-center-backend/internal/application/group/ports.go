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
