package contest

import "context"

// TeamSelectionChecker reports whether a user is already a selected member
// in any team registration for the given contest (contest_team_participants).
type TeamSelectionChecker interface {
	IsUserSelectedInAnyTeam(ctx context.Context, contestID string, userID string) (bool, error)
}
