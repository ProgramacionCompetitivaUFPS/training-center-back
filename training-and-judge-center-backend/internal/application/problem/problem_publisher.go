package problem

import (
	"context"
	"time"
)

type ProblemPublisher interface {
	MarkPublished(ctx context.Context, problemID string, now time.Time) error
}
