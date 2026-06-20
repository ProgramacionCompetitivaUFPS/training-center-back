package submission

import (
	"context"
	"time"
)

// RabbitMQ delivers higher numeric priorities first (x-max-priority: 4).
// Contest submissions must be processed before practice and bulk rejudges.
const (
	QueuePriorityContest     = 4
	QueuePriorityPostContest = 3
	QueuePriorityPractice    = 2
	QueuePriorityRejudge     = 1
)

type SubmissionQueueMetadata struct {
	ContestID *string
	ProblemID string
	UserID    string
	Language  string
}

type SubmissionQueueMessage struct {
	SubmissionID string
	Priority     int
	EnqueuedAt   time.Time
	Metadata     SubmissionQueueMetadata
}

type SubmissionQueue interface {
	Publish(ctx context.Context, msg SubmissionQueueMessage) error
}
