package team

import "context"

// RegistrationAtRisk is a team registration in a contest that has not started and that a member leaving
// the team would leave below the contest's minimum team size.
type RegistrationAtRisk struct {
	ContestID   string
	ContestName string
	GroupID     string
}

type ScheduledParticipationCleaner interface {
	// RemoveUserFromScheduledParticipations takes userID out of the selected members of every team
	// registration in a contest that has not started. A registration that would be left with fewer
	// selected members than the contest's minimum team size is cancelled instead, and returned.
	RemoveUserFromScheduledParticipations(ctx context.Context, teamID, userID string) ([]RegistrationAtRisk, error)

	// RegistrationsLosingMinimum lists what RemoveUserFromScheduledParticipations would cancel,
	// without changing anything.
	RegistrationsLosingMinimum(ctx context.Context, teamID, userID string) ([]RegistrationAtRisk, error)
}
