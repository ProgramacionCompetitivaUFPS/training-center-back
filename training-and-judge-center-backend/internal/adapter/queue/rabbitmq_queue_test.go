//go:build integration

package queue_test

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	adapterqueue "github.com/training-judge-center/backend/internal/adapter/queue"
	appsubmission "github.com/training-judge-center/backend/internal/application/submission"
)

func startRabbitMQ(t *testing.T) string {
	t.Helper()
	ctx := context.Background()
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "rabbitmq:3-alpine",
			ExposedPorts: []string{"5672/tcp"},
			WaitingFor:   wait.ForListeningPort("5672/tcp").WithStartupTimeout(60 * time.Second),
		},
		Started: true,
	})
	if err != nil {
		t.Fatalf("start rabbitmq container: %v", err)
	}
	t.Cleanup(func() { _ = container.Terminate(ctx) })

	host, _ := container.Host(ctx)
	port, _ := container.MappedPort(ctx, "5672")
	return fmt.Sprintf("amqp://guest:guest@%s:%s/", host, port.Port())
}

func consume(t *testing.T, url string) []byte {
	t.Helper()
	conn, err := amqp.Dial(url)
	if err != nil {
		t.Fatalf("consume dial: %v", err)
	}
	defer conn.Close()
	ch, err := conn.Channel()
	if err != nil {
		t.Fatalf("consume channel: %v", err)
	}
	defer ch.Close()

	msgs, err := ch.Consume("submissions", "", true, false, false, false, nil)
	if err != nil {
		t.Fatalf("consume: %v", err)
	}
	select {
	case msg := <-msgs:
		return msg.Body
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for message")
		return nil
	}
}

func TestPublish_DeliversPriorityMessage(t *testing.T) {
	url := startRabbitMQ(t)

	q, err := adapterqueue.NewRabbitMQQueue(url)
	if err != nil {
		t.Fatalf("NewRabbitMQQueue: %v", err)
	}
	defer q.Close()

	contestID := "contest-abc"
	msg := appsubmission.SubmissionQueueMessage{
		SubmissionID: "sub-001",
		Priority:     appsubmission.QueuePriorityContest,
		EnqueuedAt:   time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC),
		Metadata: appsubmission.SubmissionQueueMetadata{
			ContestID: &contestID,
			ProblemID: "prob-001",
			UserID:    "user-001",
			Language:  "cpp20",
		},
	}

	if err := q.Publish(context.Background(), msg); err != nil {
		t.Fatalf("Publish: %v", err)
	}

	body := consume(t, url)

	var got map[string]interface{}
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if got["submissionId"] != "sub-001" {
		t.Errorf("submissionId = %v, want sub-001", got["submissionId"])
	}
	if got["priority"] != float64(appsubmission.QueuePriorityContest) {
		t.Errorf("priority = %v, want %d", got["priority"], appsubmission.QueuePriorityContest)
	}
	if got["enqueuedAt"] != "2026-01-15T12:00:00Z" {
		t.Errorf("enqueuedAt = %v, want 2026-01-15T12:00:00Z", got["enqueuedAt"])
	}
	meta, _ := got["metadata"].(map[string]interface{})
	if meta == nil {
		t.Fatal("metadata missing")
	}
	if meta["contestId"] != "contest-abc" {
		t.Errorf("metadata.contestId = %v, want contest-abc", meta["contestId"])
	}
	if meta["problemId"] != "prob-001" {
		t.Errorf("metadata.problemId = %v, want prob-001", meta["problemId"])
	}
	if meta["userId"] != "user-001" {
		t.Errorf("metadata.userId = %v, want user-001", meta["userId"])
	}
	if meta["language"] != "cpp20" {
		t.Errorf("metadata.language = %v, want cpp20", meta["language"])
	}
}

func TestPublish_NilContestID_OmitsField(t *testing.T) {
	url := startRabbitMQ(t)

	q, err := adapterqueue.NewRabbitMQQueue(url)
	if err != nil {
		t.Fatalf("NewRabbitMQQueue: %v", err)
	}
	defer q.Close()

	msg := appsubmission.SubmissionQueueMessage{
		SubmissionID: "sub-002",
		Priority:     appsubmission.QueuePriorityPractice,
		EnqueuedAt:   time.Now(),
		Metadata: appsubmission.SubmissionQueueMetadata{
			ContestID: nil,
			ProblemID: "prob-002",
			UserID:    "user-002",
			Language:  "python310",
		},
	}

	if err := q.Publish(context.Background(), msg); err != nil {
		t.Fatalf("Publish: %v", err)
	}

	body := consume(t, url)

	var got map[string]interface{}
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	meta, _ := got["metadata"].(map[string]interface{})
	if _, exists := meta["contestId"]; exists {
		t.Error("contestId should be omitted when nil")
	}
	if got["submissionId"] != "sub-002" {
		t.Errorf("submissionId = %v, want sub-002", got["submissionId"])
	}
}
