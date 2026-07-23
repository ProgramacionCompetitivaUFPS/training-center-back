package contest

import "context"

// TeamParticipantProvider provides team participation data for standings computation.
type TeamParticipantProvider interface {
	// ListSelectedMembersByContest returns map[teamID][]memberUserID for all
	// team registrations in a contest. Returns an empty map when none exist.
	ListSelectedMembersByContest(ctx context.Context, contestID string) (map[string][]string, error)
}
