package material

import "context"

// AuthorIDProvider resolves a user nickname to their internal ID for author filtering.
type AuthorIDProvider interface {
	FindIDByNickname(ctx context.Context, nickname string) (string, bool, error)
}
