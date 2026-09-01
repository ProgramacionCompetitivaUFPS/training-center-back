package queue

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	appsubmission "github.com/training-judge-center/backend/internal/application/submission"
	"github.com/training-judge-center/backend/pkg/apperror"
)

// RabbitMQSubmissionQueue implements application/submission.SubmissionQueue by
// wrapping the connection RabbitMQQueue manages — see RabbitMQValidationQueue
// for why this is a sibling adapter instead of a method on RabbitMQQueue.
type RabbitMQSubmissionQueue struct {
	q *RabbitMQQueue
}

func NewRabbitMQSubmissionQueue(q *RabbitMQQueue) *RabbitMQSubmissionQueue {
	return &RabbitMQSubmissionQueue{q: q}
}

func (s *RabbitMQSubmissionQueue) Publish(ctx context.Context, msg appsubmission.SubmissionQueueMessage) error {
	body, err := json.Marshal(queueMessage{
		SubmissionID: msg.SubmissionID,
		Priority:     msg.Priority,
		EnqueuedAt:   msg.EnqueuedAt.UTC().Format(time.RFC3339),
		Metadata: queueMetadata{
			ContestID: msg.Metadata.ContestID,
			ProblemID: msg.Metadata.ProblemID,
			UserID:    msg.Metadata.UserID,
			Language:  msg.Metadata.Language,
		},
	})
	if err != nil {
		slog.ErrorContext(ctx, "rabbitmq: marshal message", "error", err)
		return apperror.NewInternal()
	}
	return s.q.publishEnvelope(ctx, kindSubmission, body, uint8(msg.Priority))
}

// submissionPayloadHandler implements payloadHandler for submission messages.
type submissionPayloadHandler struct {
	handler func(ctx context.Context, msg appsubmission.SubmissionQueueMessage) error
}

// NewSubmissionPayloadHandler builds the payloadHandler that Consume needs to
// dispatch submission messages — pass its result straight into Consume.
func NewSubmissionPayloadHandler(handler func(ctx context.Context, msg appsubmission.SubmissionQueueMessage) error) payloadHandler {
	return submissionPayloadHandler{handler: handler}
}

func (h submissionPayloadHandler) kind() messageKind { return kindSubmission }

func (h submissionPayloadHandler) handle(ctx context.Context, payload json.RawMessage) error {
	msg, err := parseSubmissionPayload(payload)
	if err != nil {
		return err
	}
	return h.handler(ctx, msg)
}

// parseSubmissionPayload converts a queueEnvelope's payload into a
// SubmissionQueueMessage — the wire-format knowledge for this queue kind
// lives here, not in the shared consume loop.
func parseSubmissionPayload(payload json.RawMessage) (appsubmission.SubmissionQueueMessage, error) {
	var raw queueMessage
	if err := json.Unmarshal(payload, &raw); err != nil {
		return appsubmission.SubmissionQueueMessage{}, err
	}
	enqueuedAt, err := time.Parse(time.RFC3339, raw.EnqueuedAt)
	if err != nil {
		return appsubmission.SubmissionQueueMessage{}, err
	}
	return appsubmission.SubmissionQueueMessage{
		SubmissionID: raw.SubmissionID,
		Priority:     raw.Priority,
		EnqueuedAt:   enqueuedAt,
		Metadata: appsubmission.SubmissionQueueMetadata{
			ContestID: raw.Metadata.ContestID,
			ProblemID: raw.Metadata.ProblemID,
			UserID:    raw.Metadata.UserID,
			Language:  raw.Metadata.Language,
		},
	}, nil
}
