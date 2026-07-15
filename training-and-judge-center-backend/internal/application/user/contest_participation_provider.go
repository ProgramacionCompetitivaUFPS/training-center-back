package user

import "context"

// ContestParticipationProvider computes the count of distinct contests the
// user has ever registered to, regardless of contest status.
type ContestParticipationProvider interface {
	// GetContestsParticipatedCount returns the count of distinct contests the
	// user is or was registered to, individually or as part of a team.
	GetContestsParticipatedCount(ctx context.Context, userID string) (int, error)
}
