package queue

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	appproblem "github.com/training-judge-center/backend/internal/application/problem"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type validationQueueMessage struct {
	ValidationID string `json:"validationId"`
	ProblemID    string `json:"problemId"`
	Slug         string `json:"slug"`
	Priority     int    `json:"priority"`
	EnqueuedAt   string `json:"enqueuedAt"`
}

// RabbitMQValidationQueue implements application/problem.ValidationQueue by
// wrapping the connection RabbitMQQueue manages — Go doesn't allow two
// methods named Publish with different signatures on one type, so both this
// and RabbitMQSubmissionQueue are thin sibling adapters over the same
// connection, not separate AMQP connections.
type RabbitMQValidationQueue struct {
	q *RabbitMQQueue
}

func NewRabbitMQValidationQueue(q *RabbitMQQueue) *RabbitMQValidationQueue {
	return &RabbitMQValidationQueue{q: q}
}

func (v *RabbitMQValidationQueue) Publish(ctx context.Context, msg appproblem.ValidationQueueMessage) error {
	body, err := json.Marshal(validationQueueMessage{
		ValidationID: msg.ValidationID,
		ProblemID:    msg.ProblemID,
		Slug:         msg.Slug,
		Priority:     msg.Priority,
		EnqueuedAt:   msg.EnqueuedAt.UTC().Format(time.RFC3339),
	})
	if err != nil {
		slog.ErrorContext(ctx, "rabbitmq: marshal validation message", "error", err)
		return apperror.NewInternal()
	}
	return v.q.publishEnvelope(ctx, kindProblemValidation, body, uint8(msg.Priority))
}

// validationPayloadHandler implements payloadHandler for validation messages.
type validationPayloadHandler struct {
	handler func(ctx context.Context, msg appproblem.ValidationQueueMessage) error
}

// NewValidationPayloadHandler builds the payloadHandler that Consume needs to
// dispatch validation messages — pass its result straight into Consume.
func NewValidationPayloadHandler(handler func(ctx context.Context, msg appproblem.ValidationQueueMessage) error) payloadHandler {
	return validationPayloadHandler{handler: handler}
}

func (h validationPayloadHandler) kind() messageKind { return kindProblemValidation }

func (h validationPayloadHandler) handle(ctx context.Context, payload json.RawMessage) error {
	msg, err := parseValidationPayload(payload)
	if err != nil {
		return err
	}
	return h.handler(ctx, msg)
}

// parseValidationPayload converts a queueEnvelope's payload into a
// ValidationQueueMessage — the wire-format knowledge for this queue kind
// lives here, not in the shared consume loop.
func parseValidationPayload(payload json.RawMessage) (appproblem.ValidationQueueMessage, error) {
	var raw validationQueueMessage
	if err := json.Unmarshal(payload, &raw); err != nil {
		return appproblem.ValidationQueueMessage{}, err
	}
	enqueuedAt, err := time.Parse(time.RFC3339, raw.EnqueuedAt)
	if err != nil {
		return appproblem.ValidationQueueMessage{}, err
	}
	return appproblem.ValidationQueueMessage{
		ValidationID: raw.ValidationID,
		ProblemID:    raw.ProblemID,
		Slug:         raw.Slug,
		Priority:     raw.Priority,
		EnqueuedAt:   enqueuedAt,
	}, nil
}
