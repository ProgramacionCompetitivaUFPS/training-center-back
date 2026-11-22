package queue

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"

	amqp "github.com/rabbitmq/amqp091-go"
	appsubmission "github.com/training-judge-center/backend/internal/application/submission"
)

func envelopeBody(t *testing.T, kind messageKind, payload json.RawMessage) []byte {
	t.Helper()
	body, err := json.Marshal(queueEnvelope{Kind: kind, Payload: payload})
	if err != nil {
		t.Fatalf("marshal envelope: %v", err)
	}
	return body
}

func TestHandleDelivery(t *testing.T) {
	validPayload, err := json.Marshal(queueMessage{
		SubmissionID: "sub-001",
		Priority:     4,
		EnqueuedAt:   "2026-01-15T12:00:00Z",
		Metadata:     queueMetadata{ProblemID: "prob-001", UserID: "user-001", Language: "cpp20"},
	})
	if err != nil {
		t.Fatalf("marshal fixture: %v", err)
	}

	tests := []struct {
		name        string
		body        []byte
		handlerErr  error
		wantCalls   []ackCall
		wantHandled bool
	}{
		{
			name:        "a message the handler accepts is acknowledged",
			body:        envelopeBody(t, kindSubmission, validPayload),
			wantCalls:   []ackCall{{method: "ack"}},
			wantHandled: true,
		},
		{
			name:        "a message the handler fails on is rejected without requeue",
			body:        envelopeBody(t, kindSubmission, validPayload),
			handlerErr:  errHandlerFailed,
			wantCalls:   []ackCall{{method: "nack"}},
			wantHandled: true,
		},
		{
			name:      "a body that is not JSON is rejected without requeue",
			body:      []byte("not json"),
			wantCalls: []ackCall{{method: "nack"}},
		},
		{
			name:      "an unknown message kind is rejected without requeue",
			body:      []byte(`{"kind":"BANANA","payload":{}}`),
			wantCalls: []ackCall{{method: "nack"}},
		},
		{
			name:      "a kind with no registered handler is rejected without requeue",
			body:      envelopeBody(t, kindProblemValidation, json.RawMessage(`{}`)),
			wantCalls: []ackCall{{method: "nack"}},
		},
		{
			name:      "a payload that does not parse is rejected without requeue",
			body:      envelopeBody(t, kindSubmission, json.RawMessage(`{"enqueuedAt":"yesterday"}`)),
			wantCalls: []ackCall{{method: "nack"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var handled []appsubmission.SubmissionQueueMessage
			byKind := map[messageKind]payloadHandler{
				kindSubmission: NewSubmissionPayloadHandler(func(_ context.Context, msg appsubmission.SubmissionQueueMessage) error {
					handled = append(handled, msg)
					return tt.handlerErr
				}),
			}
			ack := &mockAcknowledger{}
			q := &RabbitMQQueue{}

			q.handleDelivery(context.Background(), amqp.Delivery{Acknowledger: ack, Body: tt.body}, byKind)

			if !reflect.DeepEqual(ack.calls, tt.wantCalls) {
				t.Errorf("acknowledger calls = %+v, want %+v", ack.calls, tt.wantCalls)
			}
			if got := len(handled) == 1; got != tt.wantHandled {
				t.Errorf("handler called = %v, want %v", got, tt.wantHandled)
			}
		})
	}
}

func TestHandleDelivery_PassesTheParsedMessageToTheHandler(t *testing.T) {
	payload, err := json.Marshal(queueMessage{
		SubmissionID: "sub-001",
		Priority:     4,
		EnqueuedAt:   "2026-01-15T12:00:00Z",
		Metadata:     queueMetadata{ProblemID: "prob-001", UserID: "user-001", Language: "cpp20"},
	})
	if err != nil {
		t.Fatalf("marshal fixture: %v", err)
	}
	var got appsubmission.SubmissionQueueMessage
	byKind := map[messageKind]payloadHandler{
		kindSubmission: NewSubmissionPayloadHandler(func(_ context.Context, msg appsubmission.SubmissionQueueMessage) error {
			got = msg
			return nil
		}),
	}
	q := &RabbitMQQueue{}

	q.handleDelivery(context.Background(), amqp.Delivery{Acknowledger: &mockAcknowledger{}, Body: envelopeBody(t, kindSubmission, payload)}, byKind)

	if got.SubmissionID != "sub-001" || got.Priority != 4 || got.Metadata.ProblemID != "prob-001" || got.Metadata.Language != "cpp20" {
		t.Errorf("handler received %+v", got)
	}
}
