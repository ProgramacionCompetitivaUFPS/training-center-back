package submission

import "context"

type TeamMembershipChecker interface {
	IsTeammate(ctx context.Context, contestID, viewerID, authorID string) (bool, error)
}
