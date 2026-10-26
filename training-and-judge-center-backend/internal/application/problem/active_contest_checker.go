package problem

import "context"

type ActiveContestChecker interface {
	IsProblemInActiveContest(ctx context.Context, problemID string) (bool, error)
	// HasContestSubmissions reports whether anyone submitted this problem inside a contest: its
	// standings are the official result of that contest and depend on the problem existing.
	// Inside a transaction it first locks the problem row (FOR UPDATE): a submission being inserted
	// at that moment commits before the check, and no other can be inserted until the transaction ends.
	HasContestSubmissions(ctx context.Context, problemID string) (bool, error)
}
