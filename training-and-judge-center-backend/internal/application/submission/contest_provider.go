package submission

import "context"

type ContestDisplay struct {
	ID   string
	Name string
}

type ContestProvider interface {
	GetContestByID(ctx context.Context, contestID string) (*ContestDisplay, error)
}
