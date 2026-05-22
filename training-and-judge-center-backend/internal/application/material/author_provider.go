package material

import "context"

type AuthorDisplay struct {
	Nickname string
	Name     string
}

type AuthorProvider interface {
	GetDisplays(ctx context.Context, userIDs []string) (map[string]*AuthorDisplay, error)
}
