package team

import "context"

type UserDisplay struct {
	Nickname string
}

type UserProvider interface {
	GetDisplay(ctx context.Context, userID string) (*UserDisplay, error)
}
