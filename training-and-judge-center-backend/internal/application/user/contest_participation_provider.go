package user

import "context"

type ContestParticipationProvider interface {
	// GetContestsParticipatedCount returns the count of distinct contests the
	// user is or was registered to (individually or via team), regardless of status.
	GetContestsParticipatedCount(ctx context.Context, userID string) (int, error)
}
