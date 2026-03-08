package user

import "context"

type DisplayProvider interface {
	GetDisplay(ctx context.Context, userID string) (*Display, error)
	GetDisplays(ctx context.Context, userIDs []string) (map[string]*Display, error)
}
