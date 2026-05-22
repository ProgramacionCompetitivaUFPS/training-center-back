package contest

import "context"

type ContestParticipantProvider interface {
	IsRegistered(ctx context.Context, contestID, userID string) (bool, error)
	CountParticipants(ctx context.Context, contestID string) (int, error)
	CountParticipantsBulk(ctx context.Context, contestIDs []string) (map[string]int, error)
	IsRegisteredBulk(ctx context.Context, contestIDs []string, userID string) (map[string]bool, error)
}
