package material

import "context"

type GroupProvider interface {
	Exists(ctx context.Context, groupID string) (bool, error)
}

type GroupMemberProvider interface {
	IsLeadOfGroup(ctx context.Context, userID, groupID string) (bool, error)
}
