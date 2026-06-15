package group

import "context"

type UserDisplay struct {
	ID         string
	Nickname   string
	Name       string
	Email      string
	SystemRole string
}

type UserProvider interface {
	GetDisplays(ctx context.Context, userIDs []string) (map[string]*UserDisplay, error)
}
