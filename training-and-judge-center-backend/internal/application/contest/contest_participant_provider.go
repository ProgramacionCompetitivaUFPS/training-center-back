package contest

import "context"

type ContestParticipantProvider interface {
	IsRegistered(ctx context.Context, contestID, userID string) (bool, error)
	CountParticipants(ctx context.Context, contestID string) (int, error)
}
