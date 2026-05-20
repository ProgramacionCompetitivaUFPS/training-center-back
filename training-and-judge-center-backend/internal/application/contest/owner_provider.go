package contest

import "context"

type UserDisplay struct {
	ID       string
	Nickname string
	Name     string
}

type OwnerProvider interface {
	GetDisplay(ctx context.Context, userID string) (*UserDisplay, error)
}
