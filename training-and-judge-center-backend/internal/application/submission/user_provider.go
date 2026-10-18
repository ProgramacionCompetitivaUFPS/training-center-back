package submission

import "context"

type UserDisplay struct {
	ID       string
	Nickname string
}

type UserProvider interface {
	GetUserByID(ctx context.Context, userID string) (*UserDisplay, error)
	// GetUsersByIDs batches what would otherwise be one GetUserByID call per row in a list.
	GetUsersByIDs(ctx context.Context, userIDs []string) (map[string]*UserDisplay, error)
}
