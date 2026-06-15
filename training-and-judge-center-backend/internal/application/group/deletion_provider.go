package group

import "context"

// DeletionCounts holds pre-deletion counts for all entities associated with a group.
// ContestIDs is also included so the use case can invalidate standings caches after deletion.
type DeletionCounts struct {
	ContestIDs       []string
	ContestsCount    int
	MaterialsCount   int
	SubmissionsCount int
	MembersCount     int
}

// GroupDeletionProvider queries aggregate counts across all group-owned entities
// before the group is deleted, so the use case can return a meaningful summary.
type GroupDeletionProvider interface {
	GetDeletionCounts(ctx context.Context, groupID string) (DeletionCounts, error)
}
