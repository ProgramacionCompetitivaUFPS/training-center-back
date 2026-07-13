package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	appsubmission "github.com/training-judge-center/backend/internal/application/submission"
	"github.com/training-judge-center/backend/pkg/apperror"
)

const (
	submissionQueueName = "submissions"
	maxPriority         = 4
)

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

func (q *RabbitMQQueue) Publish(ctx context.Context, msg appsubmission.SubmissionQueueMessage) error {
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

	q.mu.Lock()
	defer q.mu.Unlock()

	if err := q.publishLocked(body, uint8(msg.Priority)); err != nil {
		slog.WarnContext(ctx, "rabbitmq: publish failed, reconnecting", "error", err)
		if reconErr := q.connect(); reconErr != nil {
			slog.ErrorContext(ctx, "rabbitmq: reconnect failed", "error", reconErr)
			return apperror.NewInternal()
		}
		if err := q.publishLocked(body, uint8(msg.Priority)); err != nil {
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

func (q *RabbitMQQueue) Consume(ctx context.Context, handler func(ctx context.Context, msg appsubmission.SubmissionQueueMessage) error) error {
	q.mu.Lock()
	deliveries, err := q.ch.Consume(submissionQueueName, "", false, false, false, false, nil)
	q.mu.Unlock()
	if err != nil {
		return fmt.Errorf("rabbitmq: consume: %w", err)
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		case d, ok := <-deliveries:
			if !ok {
				return fmt.Errorf("rabbitmq: deliveries channel closed")
			}
			var raw queueMessage
			if err := json.Unmarshal(d.Body, &raw); err != nil {
				slog.ErrorContext(ctx, "rabbitmq: failed to unmarshal message", "error", err)
				_ = d.Nack(false, false)
				continue
			}
			enqueuedAt, err := time.Parse(time.RFC3339, raw.EnqueuedAt)
			if err != nil {
				slog.ErrorContext(ctx, "rabbitmq: failed to parse enqueuedAt", "raw", raw.EnqueuedAt, "error", err)
				_ = d.Nack(false, false)
				continue
			}
			msg := appsubmission.SubmissionQueueMessage{
				SubmissionID: raw.SubmissionID,
				Priority:     raw.Priority,
				EnqueuedAt:   enqueuedAt,
				Metadata: appsubmission.SubmissionQueueMetadata{
					ContestID: raw.Metadata.ContestID,
					ProblemID: raw.Metadata.ProblemID,
					UserID:    raw.Metadata.UserID,
					Language:  raw.Metadata.Language,
				},
			}
			if err := handler(ctx, msg); err != nil {
				slog.ErrorContext(ctx, "rabbitmq: handler returned error", "error", err)
				_ = d.Nack(false, false)
				continue
			}
			_ = d.Ack(false)
		}
	}
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
