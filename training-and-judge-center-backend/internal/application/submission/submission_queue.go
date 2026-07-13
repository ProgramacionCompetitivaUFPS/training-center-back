package submission

import (
	"context"
	"time"
)

// Single "submissions" queue with x-max-priority: 4 instead of separate per-type queues — avoids per-queue workers and manual routing.
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
