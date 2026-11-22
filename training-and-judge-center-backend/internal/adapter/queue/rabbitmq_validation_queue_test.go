package queue

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	appproblem "github.com/training-judge-center/backend/internal/application/problem"
)

func TestParseValidationPayload_ValidPayload(t *testing.T) {
	raw, err := json.Marshal(validationQueueMessage{
		ValidationID: "validation-001",
		ProblemID:    "problem-001",
		Slug:         "sum-of-two-numbers",
		Priority:     5,
		EnqueuedAt:   "2026-01-15T12:00:00Z",
	})
	if err != nil {
		t.Fatalf("marshal fixture: %v", err)
	}

	msg, err := parseValidationPayload(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if msg.ValidationID != "validation-001" {
		t.Errorf("ValidationID: got %q, want validation-001", msg.ValidationID)
	}
	if msg.ProblemID != "problem-001" {
		t.Errorf("ProblemID: got %q, want problem-001", msg.ProblemID)
	}
	if msg.Slug != "sum-of-two-numbers" {
		t.Errorf("Slug: got %q, want sum-of-two-numbers", msg.Slug)
	}
	if msg.Priority != 5 {
		t.Errorf("Priority: got %d, want 5", msg.Priority)
	}
	wantTime := time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC)
	if !msg.EnqueuedAt.Equal(wantTime) {
		t.Errorf("EnqueuedAt: got %v, want %v", msg.EnqueuedAt, wantTime)
	}
}

func TestParseValidationPayload_MalformedJSON_ReturnsError(t *testing.T) {
	_, err := parseValidationPayload(json.RawMessage(`not-json`))
	if err == nil {
		t.Error("expected error, got nil")
	}
}

func TestParseValidationPayload_InvalidEnqueuedAt_ReturnsError(t *testing.T) {
	raw, err := json.Marshal(validationQueueMessage{
		ValidationID: "validation-002",
		ProblemID:    "problem-002",
		Priority:     5,
		EnqueuedAt:   "not-a-timestamp",
	})
	if err != nil {
		t.Fatalf("marshal fixture: %v", err)
	}

	_, err = parseValidationPayload(raw)
	if err == nil {
		t.Error("expected error, got nil")
	}
}

func TestValidationPayloadHandler_Handle_PassesTheParsedMessageToTheHandler(t *testing.T) {
	raw, err := json.Marshal(validationQueueMessage{
		ValidationID: "validation-001",
		ProblemID:    "problem-001",
		Slug:         "sum-of-two-numbers",
		Priority:     5,
		EnqueuedAt:   "2026-01-15T12:00:00Z",
	})
	if err != nil {
		t.Fatalf("marshal fixture: %v", err)
	}
	var got appproblem.ValidationQueueMessage
	h := NewValidationPayloadHandler(func(_ context.Context, msg appproblem.ValidationQueueMessage) error {
		got = msg
		return nil
	})

	err = h.handle(context.Background(), raw)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if h.kind() != kindProblemValidation {
		t.Errorf("kind: got %v, want %v", h.kind(), kindProblemValidation)
	}
	if got.ValidationID != "validation-001" || got.Slug != "sum-of-two-numbers" || got.Priority != 5 {
		t.Errorf("handler received %+v", got)
	}
}

func TestValidationPayloadHandler_Handle_MalformedPayload_ReturnsErrorWithoutCallingTheHandler(t *testing.T) {
	called := false
	h := NewValidationPayloadHandler(func(context.Context, appproblem.ValidationQueueMessage) error {
		called = true
		return nil
	})

	err := h.handle(context.Background(), json.RawMessage(`{"enqueuedAt":"yesterday"}`))

	if err == nil {
		t.Error("expected an error for a payload that does not parse")
	}
	if called {
		t.Error("the handler must not run for a payload that does not parse")
	}
}

func TestValidationPayloadHandler_Handle_HandlerError_IsReturned(t *testing.T) {
	raw, _ := json.Marshal(validationQueueMessage{ValidationID: "validation-001", EnqueuedAt: "2026-01-15T12:00:00Z"})
	h := NewValidationPayloadHandler(func(context.Context, appproblem.ValidationQueueMessage) error {
		return errHandlerFailed
	})

	err := h.handle(context.Background(), raw)

	if !errors.Is(err, errHandlerFailed) {
		t.Errorf("got %v, want the handler error", err)
	}
}
