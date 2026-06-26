package group

import "context"

type TeamSelectionCleaner interface {
	RemoveFromScheduledByGroupAndUser(ctx context.Context, groupID, userID string) error
}
