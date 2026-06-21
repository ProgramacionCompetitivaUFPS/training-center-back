package contest

import "context"

// CallerStandingProvider resolves the caller's standing ID in a contest.
// Individual → standingID = userID; team member → standingID = teamID.
type CallerStandingProvider interface {
	GetCallerStandingID(ctx context.Context, contestID, userID string) (standingID string, found bool, err error)
}
