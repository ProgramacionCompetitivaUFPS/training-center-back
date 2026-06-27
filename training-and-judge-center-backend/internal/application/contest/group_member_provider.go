package contest

import "context"

type GroupMemberProvider interface {
	// GetMemberRole returns the caller's role in the group (e.g. "LEAD", "MEMBER"),
	// or nil if the user is not a member at all.
	GetMemberRole(ctx context.Context, userID, groupID string) (*string, error)
}
