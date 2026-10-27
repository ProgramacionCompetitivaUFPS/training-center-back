package problem

import (
	"context"
	"time"
)

type ProblemPublisher interface {
	MarkPublished(ctx context.Context, problemID string, now time.Time) error
	// MarkPublishedIfUnchanged publishes the draft only if its updated_at is still unchangedSince, so an
	// edit saved after the caller read the problem is never published on the strength of an older
	// validation. It reports false, with no error, when the problem changed (or is no longer a draft).
	MarkPublishedIfUnchanged(ctx context.Context, problemID string, unchangedSince, now time.Time) (bool, error)
}
