package material

import "context"

type GroupProvider interface {
	Exists(ctx context.Context, groupID string) (bool, error)
}
