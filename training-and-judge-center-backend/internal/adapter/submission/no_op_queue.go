package submission

import (
	"context"

	appSubmission "github.com/training-judge-center/backend/internal/application/submission"
)

// NoOpQueue is a placeholder until S-5 wires the real RabbitMQ publisher.
type NoOpQueue struct{}

func (NoOpQueue) Publish(_ context.Context, _ appSubmission.SubmissionQueueMessage) error {
	return nil
}
