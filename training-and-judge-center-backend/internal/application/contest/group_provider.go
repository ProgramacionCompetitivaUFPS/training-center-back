package contest

import "context"

type GroupInfo struct {
	ID   string
	Name string
}

type GroupProvider interface {
	FindByID(ctx context.Context, groupID string) (*GroupInfo, error)
}
