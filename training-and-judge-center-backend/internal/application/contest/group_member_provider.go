package contest

import "context"

type GroupMemberProvider interface {
	// GetMemberRole returns the caller's role in the group (e.g. "LEAD", "MEMBER"),
	// or nil if the user is not a member at all.
	GetMemberRole(ctx context.Context, userID, groupID string) (*string, error)
	// GetMemberRoleBulk returns a map[groupID]*role for all groups where the user
	// is a member. Groups where the user has no membership are absent from the map.
	GetMemberRoleBulk(ctx context.Context, userID string, groupIDs []string) (map[string]*string, error)
}
