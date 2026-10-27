package queue

import (
	"encoding/json"
	"testing"
	"time"
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
