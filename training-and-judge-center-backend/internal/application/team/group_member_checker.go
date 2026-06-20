package team

import "context"

// GroupMemberChecker reports which of the given user IDs are members of the
// given group.
type GroupMemberChecker interface {
	AreUsersGroupMembers(ctx context.Context, groupID string, userIDs []string) (map[string]bool, error)
}
