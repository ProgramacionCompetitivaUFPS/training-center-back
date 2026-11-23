package user

import "context"

// TeamMembershipCleaner takes a user out of every team they belong to. It runs when an account is
// deactivated: the person can no longer sign in, so nobody else could ever remove them, and they
// would stay on the roster (and in its count) under an anonymized nickname.
type TeamMembershipCleaner interface {
	// RemoveUserFromAllTeams deletes the user's team memberships and pending team invitations, and
	// takes them out of the registrations of contests that have not started (cancelling the ones
	// left below the contest's minimum team size). Registrations of contests already running or
	// finished keep their selected members, so the standings stay as they were.
	RemoveUserFromAllTeams(ctx context.Context, userID string) error
}
