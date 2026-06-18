package team

import "context"

// IndividualRegistrationChecker reports which of the given user IDs are already
// registered individually (contest_registrations) to the given contest.
type IndividualRegistrationChecker interface {
	AreUsersRegisteredIndividually(ctx context.Context, contestID string, userIDs []string) (map[string]bool, error)
}
