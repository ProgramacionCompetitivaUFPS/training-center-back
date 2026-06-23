package problem

import "context"

type ActiveContestChecker interface {
	IsProblemInActiveContest(ctx context.Context, problemID string) (bool, error)
}
