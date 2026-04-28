package material

import "context"

type GroupVisibility string

const (
	GroupVisibilityVisible    GroupVisibility = "VISIBLE"
	GroupVisibilityNotVisible GroupVisibility = "NOT_VISIBLE"
)

type GroupProvider interface {
	Exists(ctx context.Context, groupID string) (bool, error)
}

type GroupVisibilityProvider interface {
	// FindVisibility returns ("VISIBLE"|"NOT_VISIBLE", true, nil) if the group exists,
	// ("", false, nil) if it does not, or ("", false, err) on infrastructure failure.
	FindVisibility(ctx context.Context, groupID string) (GroupVisibility, bool, error)
}

type GroupMemberProvider interface {
	IsLeadOfGroup(ctx context.Context, userID, groupID string) (bool, error)
	IsMemberOfGroup(ctx context.Context, userID, groupID string) (bool, error)
}

type AuthorProvider interface {
	GetDisplays(ctx context.Context, userIDs []string) (map[string]*AuthorDisplay, error)
}

// AuthorDisplay is the single author type used both as the port return DTO
// and as the use-case output DTO — no separate AuthorData needed.
type AuthorDisplay struct {
	Nickname string
	Name     string
}
