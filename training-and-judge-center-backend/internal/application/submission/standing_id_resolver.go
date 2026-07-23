package submission

import "context"

type StandingIDResolver interface {
	ResolveStandingID(ctx context.Context, contestID, userID string) (standingID string, registered bool, err error)
}
