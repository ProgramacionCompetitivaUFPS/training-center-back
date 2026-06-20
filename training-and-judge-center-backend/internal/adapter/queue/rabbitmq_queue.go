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

func (q *RabbitMQQueue) Publish(_ context.Context, msg appsubmission.SubmissionQueueMessage) error {
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
		slog.Error("rabbitmq: marshal message", "error", err)
		return apperror.NewInternal()
	}

	q.mu.Lock()
	defer q.mu.Unlock()

	if err := q.publishLocked(body, uint8(msg.Priority)); err != nil {
		slog.Warn("rabbitmq: publish failed, reconnecting", "error", err)
		if reconErr := q.connect(); reconErr != nil {
			slog.Error("rabbitmq: reconnect failed", "error", reconErr)
			return apperror.NewInternal()
		}
		if err := q.publishLocked(body, uint8(msg.Priority)); err != nil {
			slog.Error("rabbitmq: publish failed after reconnect", "error", err)
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
