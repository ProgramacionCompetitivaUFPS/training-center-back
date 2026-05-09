package group

import "context"

type UserDisplay struct {
	Nickname string
	Name     string
	Email    string
}

type UserProvider interface {
	GetDisplays(ctx context.Context, userIDs []string) (map[string]*UserDisplay, error)
}
