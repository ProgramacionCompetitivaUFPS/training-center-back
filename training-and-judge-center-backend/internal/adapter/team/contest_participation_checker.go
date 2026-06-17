package team

import "context"

// NoOpContestParticipationChecker always returns false — the contest-team
// integration is not yet implemented (pending Register Team to Contest feature).
type NoOpContestParticipationChecker struct{}

func (c *NoOpContestParticipationChecker) IsUserInActiveContestForTeam(_ context.Context, _, _ string) (bool, error) {
	return false, nil
}
