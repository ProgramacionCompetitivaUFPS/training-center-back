package submission

import "context"

type LeadChecker interface {
	IsLeadOfContestGroup(ctx context.Context, contestID, userID string) (bool, error)
}
