package problem

import (
	"context"
	"time"
)

// QueuePriorityPublishValidation is the same tier as a contest submission, so the two are served in
// the order they arrive. It used to sit above it (5), which let a validation jump ahead of every
// contest submission already waiting. It still beats practice, post-contest and rejudging, which is
// where someone is blocked on the same HTTP connection waiting for the result.
//
// Keep it equal to submission.QueuePriorityContest: the problem package cannot import that one, and
// a test in adapter/queue holds the two together.
const QueuePriorityPublishValidation = 4

type ValidationQueueMessage struct {
	ValidationID string
	ProblemID    string
	// Slug is carried along only for building a readable storage path for
	// compiled checker/validator artifacts — the ticket itself is looked up
	// by ValidationID, never by slug.
	Slug       string
	Priority   int
	EnqueuedAt time.Time
}

type ValidationQueue interface {
	Publish(ctx context.Context, msg ValidationQueueMessage) error
}
