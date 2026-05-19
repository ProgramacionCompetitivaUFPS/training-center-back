package contest

import "context"

type GroupMemberProvider interface {
	IsLeadOfGroup(ctx context.Context, userID, groupID string) (bool, error)
}
