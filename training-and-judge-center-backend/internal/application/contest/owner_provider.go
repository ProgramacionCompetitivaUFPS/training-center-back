package contest

import "context"

type UserDisplay struct {
	Nickname string
	Name     string
}

type OwnerProvider interface {
	GetDisplay(ctx context.Context, userID string) (*UserDisplay, error)
}
