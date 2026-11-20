package submission

import "context"

type GroupMembershipChecker interface {
	IsGroupMember(ctx context.Context, groupID, userID string) (bool, error)
}
