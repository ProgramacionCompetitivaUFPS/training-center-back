package problem

import (
	"context"

	appProblem "github.com/training-judge-center/backend/internal/application/problem"
)

// NoOpValidationQueue is used when RABBITMQ_URL is not set (local dev).
type NoOpValidationQueue struct{}

func (NoOpValidationQueue) Publish(_ context.Context, _ appProblem.ValidationQueueMessage) error {
	return nil
}
