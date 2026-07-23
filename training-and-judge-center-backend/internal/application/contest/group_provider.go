package contest

import "context"

type GroupInfo struct {
	ID        string
	Name      string
	IsVisible bool
}

type GroupProvider interface {
	FindByID(ctx context.Context, groupID string) (*GroupInfo, error)
	// FindByIDs bulk-resolves group display info for a page of contests,
	// avoiding N+1 lookups. Missing IDs are simply absent from the map.
	FindByIDs(ctx context.Context, groupIDs []string) (map[string]*GroupInfo, error)
	// ListAccessibleGroupIDs returns the IDs of groups the viewer can see
	// (VISIBLE groups, or groups they're a member of). isAdmin == true
	// returns nil as a sentinel meaning "no restriction" (all groups).
	ListAccessibleGroupIDs(ctx context.Context, viewerID string, isAdmin bool) ([]string, error)
}
