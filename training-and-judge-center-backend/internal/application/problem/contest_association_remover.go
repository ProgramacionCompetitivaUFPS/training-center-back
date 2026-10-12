package problem

import "context"

type ContestAssociationRemover interface {
	RemoveProblemFromContests(ctx context.Context, problemID string) error
}
