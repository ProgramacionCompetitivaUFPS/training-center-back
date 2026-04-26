package material

import "context"

type GroupProvider interface {
	Exists(ctx context.Context, groupID string) (bool, error)
}

type GroupVisibilityProvider interface {
	// FindVisibility returns ("VISIBLE"|"NOT_VISIBLE", true, nil) if the group exists,
	// or ("", false, nil) if it does not.
	FindVisibility(ctx context.Context, groupID string) (string, bool, error)
}

type GroupMemberProvider interface {
	IsLeadOfGroup(ctx context.Context, userID, groupID string) (bool, error)
	IsMemberOfGroup(ctx context.Context, userID, groupID string) (bool, error)
}

type AuthorProvider interface {
	GetDisplays(ctx context.Context, userIDs []string) (map[string]*AuthorDisplay, error)
}

type AuthorDisplay struct {
	Nickname string
	Name     string
}
