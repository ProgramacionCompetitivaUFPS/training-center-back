package problem

import (
	"context"
	"time"
)

type ContestRef struct {
	ID        string
	Name      string
	GroupID   string
	StartTime time.Time
	EndTime   time.Time
}

// AffectedContestsProvider lists the contests a problem is part of that have not
// finished yet — the ones where a change to its judging files shows up.
type AffectedContestsProvider interface {
	ListOpenContestsOfProblem(ctx context.Context, problemID string) ([]ContestRef, error)
}
