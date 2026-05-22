package material

import "context"

type GroupMemberProvider interface {
	IsLeadOfGroup(ctx context.Context, userID, groupID string) (bool, error)
	IsMemberOfGroup(ctx context.Context, userID, groupID string) (bool, error)
}
