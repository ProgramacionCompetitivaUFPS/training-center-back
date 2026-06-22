package team

import "context"

type ScheduledParticipationCleaner interface {
	RemoveUserFromScheduledParticipations(ctx context.Context, teamID, userID string) (int, error)
}
