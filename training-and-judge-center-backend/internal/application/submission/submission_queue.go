package submission

import (
	"context"
	"time"
)

const (
	QueuePriorityContest     = 1
	QueuePriorityPostContest = 2
	QueuePriorityPractice    = 3
	QueuePriorityRejudge     = 4
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
