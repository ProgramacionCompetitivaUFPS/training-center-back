package problem

import "context"

type UserDisplay struct {
	Nickname string
	Name     string
}

type UserProvider interface {
	ExistsByID(ctx context.Context, userID string) (bool, error)
	GetDisplay(ctx context.Context, userID string) (*UserDisplay, error)
	GetDisplays(ctx context.Context, userIDs []string) (map[string]*UserDisplay, error)
	GetIDByNickname(ctx context.Context, nickname string) (string, bool, error)
}
