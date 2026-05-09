package material

import "context"

// AuthorDisplay is the single author type used both as the port return DTO
// and as the use-case output DTO — no separate AuthorData needed.
type AuthorDisplay struct {
	Nickname string
	Name     string
}

type AuthorProvider interface {
	GetDisplays(ctx context.Context, userIDs []string) (map[string]*AuthorDisplay, error)
}
