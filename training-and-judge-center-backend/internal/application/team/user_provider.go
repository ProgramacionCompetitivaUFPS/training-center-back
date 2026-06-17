package team

import "context"

type UserDisplay struct {
	Nickname string
}

type UserProvider interface {
	GetDisplay(ctx context.Context, userID string) (*UserDisplay, error)
	GetDisplays(ctx context.Context, userIDs []string) (map[string]*UserDisplay, error)
}
