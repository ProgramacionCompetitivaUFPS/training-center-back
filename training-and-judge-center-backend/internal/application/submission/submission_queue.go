package submission

import "context"

const (
	QueuePriorityContest    = 1
	QueuePriorityPostContest = 2
	QueuePriorityPractice   = 3
	QueuePriorityRejudge    = 4
)

type SubmissionQueueMessage struct {
	SubmissionID string
	Priority     int
}

type SubmissionQueue interface {
	Publish(ctx context.Context, msg SubmissionQueueMessage) error
}
