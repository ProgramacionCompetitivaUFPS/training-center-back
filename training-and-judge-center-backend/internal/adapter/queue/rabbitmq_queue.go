package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/training-judge-center/backend/pkg/apperror"
)

const (
	submissionQueueName = "submissions"
	maxPriority         = 5 // must be >= appproblem.QueuePriorityPublishValidation, the highest tier on the queue
)

// queueEnvelope wraps every message published to the submissions queue so a
// single queue/consumer can carry more than one kind of job.
type queueEnvelope struct {
	Kind    messageKind     `json:"kind"`
	Payload json.RawMessage `json:"payload"`
}

// payloadHandler parses one kind of envelope payload and dispatches it to its
// own typed handler. Each message kind provides its own implementation (see
// submissionPayloadHandler, validationPayloadHandler) so Consume/handleDelivery
// never need to know those concrete message types exist.
type payloadHandler interface {
	kind() messageKind
	handle(ctx context.Context, payload json.RawMessage) error
}

// wire format published to RabbitMQ — matches RUNNER_ARCHITECTURE.md §6
type queueMessage struct {
	SubmissionID string        `json:"submissionId"`
	Priority     int           `json:"priority"`
	EnqueuedAt   string        `json:"enqueuedAt"`
	Metadata     queueMetadata `json:"metadata"`
}

type queueMetadata struct {
	ContestID *string `json:"contestId,omitempty"`
	ProblemID string  `json:"problemId"`
	UserID    string  `json:"userId"`
	Language  string  `json:"language"`
}

// RabbitMQQueue publishes submission events to a priority queue.
// It reconnects automatically on channel/connection failure.
type RabbitMQQueue struct {
	url  string
	mu   sync.Mutex
	conn *amqp.Connection
	ch   *amqp.Channel
}

func NewRabbitMQQueue(url string) (*RabbitMQQueue, error) {
	q := &RabbitMQQueue{url: url}
	if err := q.connect(); err != nil {
		return nil, err
	}
	return q, nil
}

func (q *RabbitMQQueue) connect() error {
	// Close stale resources before overwriting to avoid connection leaks.
	if q.ch != nil {
		q.ch.Close()
	}
	if q.conn != nil {
		q.conn.Close()
	}

	conn, err := amqp.Dial(q.url)
	if err != nil {
		return fmt.Errorf("rabbitmq: dial %s: %w", q.url, err)
	}
	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return fmt.Errorf("rabbitmq: open channel: %w", err)
	}
	if _, err = ch.QueueDeclare(
		submissionQueueName,
		true,  // durable
		false, // auto-delete
		false, // exclusive
		false, // no-wait
		amqp.Table{"x-max-priority": int32(maxPriority)},
	); err != nil {
		ch.Close()
		conn.Close()
		return fmt.Errorf("rabbitmq: declare queue: %w", err)
	}
	q.conn = conn
	q.ch = ch
	return nil
}

// publishEnvelope wraps payload in a queueEnvelope and publishes it, retrying
// once after a reconnect on failure. Shared by every message kind.
func (q *RabbitMQQueue) publishEnvelope(ctx context.Context, kind messageKind, payload []byte, priority uint8) error {
	body, err := json.Marshal(queueEnvelope{Kind: kind, Payload: payload})
	if err != nil {
		slog.ErrorContext(ctx, "rabbitmq: marshal envelope", "error", err)
		return apperror.NewInternal()
	}

	q.mu.Lock()
	defer q.mu.Unlock()

	if err := q.publishLocked(body, priority); err != nil {
		slog.WarnContext(ctx, "rabbitmq: publish failed, reconnecting", "error", err)
		if reconErr := q.connect(); reconErr != nil {
			slog.ErrorContext(ctx, "rabbitmq: reconnect failed", "error", reconErr)
			return apperror.NewInternal()
		}
		if err := q.publishLocked(body, priority); err != nil {
			slog.ErrorContext(ctx, "rabbitmq: publish failed after reconnect", "error", err)
			return apperror.NewInternal()
		}
	}
	return nil
}

func (q *RabbitMQQueue) publishLocked(body []byte, priority uint8) error {
	return q.ch.Publish(
		"",                  // default exchange
		submissionQueueName, // routing key
		false,               // mandatory
		false,               // immediate
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			Priority:     priority,
			Body:         body,
		},
	)
}

// Consume listens for deliveries and dispatches each one to whichever handler
// matches its kind. Passing more than one handler for the same kind is a
// programmer error — the last one silently wins, same as any Go map literal.
func (q *RabbitMQQueue) Consume(ctx context.Context, maxConcurrent int, handlers ...payloadHandler) error {
	if maxConcurrent < 1 {
		return fmt.Errorf("rabbitmq: maxConcurrent must be at least 1, got %d", maxConcurrent)
	}
	byKind := make(map[messageKind]payloadHandler, len(handlers))
	for _, h := range handlers {
		byKind[h.kind()] = h
	}
	for _, k := range allMessageKinds {
		if _, ok := byKind[k]; !ok {
			return fmt.Errorf("rabbitmq: no handler registered for message kind %q", k)
		}
	}

	q.mu.Lock()
	if err := q.ch.Qos(maxConcurrent, 0, false); err != nil {
		q.mu.Unlock()
		return fmt.Errorf("rabbitmq: qos: %w", err)
	}
	deliveries, err := q.ch.Consume(submissionQueueName, "", false, false, false, false, nil)
	q.mu.Unlock()
	if err != nil {
		return fmt.Errorf("rabbitmq: consume: %w", err)
	}

	sem := make(chan struct{}, maxConcurrent)
	var wg sync.WaitGroup

	for {
		select {
		case <-ctx.Done():
			wg.Wait()
			return nil
		case d, ok := <-deliveries:
			if !ok {
				wg.Wait()
				return fmt.Errorf("rabbitmq: deliveries channel closed")
			}
			select {
			case sem <- struct{}{}:
			case <-ctx.Done():
				_ = d.Nack(false, true)
				wg.Wait()
				return nil
			}
			wg.Add(1)
			go func(d amqp.Delivery) {
				defer func() { <-sem; wg.Done() }()
				q.handleDelivery(ctx, d, byKind)
			}(d)
		}
	}
}

func (q *RabbitMQQueue) handleDelivery(ctx context.Context, d amqp.Delivery, byKind map[messageKind]payloadHandler) {
	var env queueEnvelope
	if err := json.Unmarshal(d.Body, &env); err != nil {
		slog.ErrorContext(ctx, "rabbitmq: failed to unmarshal envelope", "error", err)
		_ = d.Nack(false, false)
		return
	}

	h, ok := byKind[env.Kind]
	if !ok {
		slog.ErrorContext(ctx, "rabbitmq: no handler registered for message kind", "kind", env.Kind)
		_ = d.Nack(false, false)
		return
	}

	if err := h.handle(ctx, env.Payload); err != nil {
		slog.ErrorContext(ctx, "rabbitmq: handler returned error", "error", err)
		_ = d.Nack(false, false)
		return
	}
	_ = d.Ack(false)
}

func (q *RabbitMQQueue) Close() {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.ch != nil {
		q.ch.Close()
	}
	if q.conn != nil {
		q.conn.Close()
	}
}
