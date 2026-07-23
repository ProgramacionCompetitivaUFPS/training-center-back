package group

import "context"

type ContestRegistrationCleaner interface {
	DeleteScheduledByGroupAndUser(ctx context.Context, groupID, userID string) (int, error)
}
