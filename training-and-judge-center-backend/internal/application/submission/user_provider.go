package submission

import "context"

type UserDisplay struct {
	ID       string
	Nickname string
}

type UserProvider interface {
	GetUserByID(ctx context.Context, userID string) (*UserDisplay, error)
}
