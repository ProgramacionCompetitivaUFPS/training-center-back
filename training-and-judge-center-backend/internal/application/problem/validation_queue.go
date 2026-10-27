package problem

import (
	"context"
	"time"
)

// QueuePriorityPublishValidation is the highest tier on the queue (above
// contest=4): it's the only place in the system where someone is blocked on
// the same HTTP connection waiting for the result.
const QueuePriorityPublishValidation = 5

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
