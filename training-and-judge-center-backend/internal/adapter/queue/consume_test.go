package queue_test

import (
	"context"
	"testing"

	adapterqueue "github.com/training-judge-center/backend/internal/adapter/queue"
	appproblem "github.com/training-judge-center/backend/internal/application/problem"
	appsubmission "github.com/training-judge-center/backend/internal/application/submission"
)

func TestConsume_InvalidMaxConcurrent_ReturnsError(t *testing.T) {
	q := &adapterqueue.RabbitMQQueue{}
	err := q.Consume(context.Background(), 0,
		adapterqueue.NewSubmissionPayloadHandler(func(ctx context.Context, msg appsubmission.SubmissionQueueMessage) error { return nil }),
		adapterqueue.NewValidationPayloadHandler(func(ctx context.Context, msg appproblem.ValidationQueueMessage) error { return nil }),
	)
	if err == nil {
		t.Error("expected error for maxConcurrent < 1, got nil")
	}
}

func TestConsume_MissingValidationHandler_ReturnsError(t *testing.T) {
	q := &adapterqueue.RabbitMQQueue{}
	err := q.Consume(context.Background(), 1,
		adapterqueue.NewSubmissionPayloadHandler(func(ctx context.Context, msg appsubmission.SubmissionQueueMessage) error { return nil }),
	)
	if err == nil {
		t.Error("expected error when the validation handler is missing, got nil")
	}
}

func TestConsume_MissingSubmissionHandler_ReturnsError(t *testing.T) {
	q := &adapterqueue.RabbitMQQueue{}
	err := q.Consume(context.Background(), 1,
		adapterqueue.NewValidationPayloadHandler(func(ctx context.Context, msg appproblem.ValidationQueueMessage) error { return nil }),
	)
	if err == nil {
		t.Error("expected error when the submission handler is missing, got nil")
	}
}

func TestConsume_NoHandlers_ReturnsError(t *testing.T) {
	q := &adapterqueue.RabbitMQQueue{}
	if err := q.Consume(context.Background(), 1); err == nil {
		t.Error("expected error when no handlers are registered, got nil")
	}
}
