package team

import "context"

type ContestParticipationChecker interface {
	IsUserInActiveContestForTeam(ctx context.Context, userID, teamID string) (bool, error)
}
