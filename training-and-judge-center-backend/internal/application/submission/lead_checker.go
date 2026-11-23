package submission

import "context"

type LeadChecker interface {
	IsLeadOfContestGroup(ctx context.Context, contestID, userID string) (bool, error)
	// IsLeadOfMember reports whether leadID leads at least one regular group that memberID
	// belongs to. The default (global) group is excluded: everyone is in it.
	IsLeadOfMember(ctx context.Context, leadID, memberID string) (bool, error)
}
