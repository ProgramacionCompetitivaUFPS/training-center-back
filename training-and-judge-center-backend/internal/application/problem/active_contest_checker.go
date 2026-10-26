package problem

import "context"

type ActiveContestChecker interface {
	IsProblemInActiveContest(ctx context.Context, problemID string) (bool, error)
	// HasContestSubmissions reports whether anyone submitted this problem inside a contest: its
	// standings are the official result of that contest and depend on the problem existing.
	HasContestSubmissions(ctx context.Context, problemID string) (bool, error)
}
