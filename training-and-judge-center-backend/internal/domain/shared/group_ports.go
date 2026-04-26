package shared

import "context"

// GroupProvider is a cross-domain port for checking group existence.
// Used by any domain that needs to verify a group exists without importing domain/group.
type GroupProvider interface {
	Exists(ctx context.Context, groupID string) (bool, error)
}

// GroupMemberProvider is a cross-domain port for checking group membership roles.
// Used by any domain that needs to verify Lead permissions without importing domain/group.
type GroupMemberProvider interface {
	IsLeadOfGroup(ctx context.Context, userID, groupID string) (bool, error)
}
