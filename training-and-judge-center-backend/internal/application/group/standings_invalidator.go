package group

import "context"

// GroupStandingsInvalidator evicts cached standings for a single contest from the
// standings store. Used by DeleteGroupUseCase after the DB transaction commits.
type GroupStandingsInvalidator interface {
	Invalidate(ctx context.Context, contestID string) error
}
