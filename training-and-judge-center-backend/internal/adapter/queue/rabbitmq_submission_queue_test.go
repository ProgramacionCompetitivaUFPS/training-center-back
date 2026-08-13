package queue

import (
	"encoding/json"
	"testing"
	"time"
)

func TestParseSubmissionPayload_ValidPayload(t *testing.T) {
	contestID := "contest-abc"
	raw, err := json.Marshal(queueMessage{
		SubmissionID: "sub-001",
		Priority:     4,
		EnqueuedAt:   "2026-01-15T12:00:00Z",
		Metadata: queueMetadata{
			ContestID: &contestID,
			ProblemID: "prob-001",
			UserID:    "user-001",
			Language:  "cpp20",
		},
	})
	if err != nil {
		t.Fatalf("marshal fixture: %v", err)
	}

	msg, err := parseSubmissionPayload(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if msg.SubmissionID != "sub-001" {
		t.Errorf("SubmissionID: got %q, want sub-001", msg.SubmissionID)
	}
	if msg.Priority != 4 {
		t.Errorf("Priority: got %d, want 4", msg.Priority)
	}
	wantTime := time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC)
	if !msg.EnqueuedAt.Equal(wantTime) {
		t.Errorf("EnqueuedAt: got %v, want %v", msg.EnqueuedAt, wantTime)
	}
	if msg.Metadata.ContestID == nil || *msg.Metadata.ContestID != "contest-abc" {
		t.Errorf("Metadata.ContestID: got %v, want contest-abc", msg.Metadata.ContestID)
	}
	if msg.Metadata.ProblemID != "prob-001" {
		t.Errorf("Metadata.ProblemID: got %q, want prob-001", msg.Metadata.ProblemID)
	}
}

func TestParseSubmissionPayload_NilContestID(t *testing.T) {
	raw, err := json.Marshal(queueMessage{
		SubmissionID: "sub-002",
		Priority:     2,
		EnqueuedAt:   "2026-01-15T12:00:00Z",
		Metadata:     queueMetadata{ContestID: nil, ProblemID: "prob-002", UserID: "user-002", Language: "python310"},
	})
	if err != nil {
		t.Fatalf("marshal fixture: %v", err)
	}

	msg, err := parseSubmissionPayload(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if msg.Metadata.ContestID != nil {
		t.Errorf("Metadata.ContestID: got %v, want nil", msg.Metadata.ContestID)
	}
}

func TestParseSubmissionPayload_MalformedJSON_ReturnsError(t *testing.T) {
	_, err := parseSubmissionPayload(json.RawMessage(`not-json`))
	if err == nil {
		t.Error("expected error, got nil")
	}
}

func TestParseSubmissionPayload_InvalidEnqueuedAt_ReturnsError(t *testing.T) {
	raw, err := json.Marshal(queueMessage{
		SubmissionID: "sub-003",
		Priority:     1,
		EnqueuedAt:   "not-a-timestamp",
		Metadata:     queueMetadata{ProblemID: "p", UserID: "u", Language: "cpp20"},
	})
	if err != nil {
		t.Fatalf("marshal fixture: %v", err)
	}

	_, err = parseSubmissionPayload(raw)
	if err == nil {
		t.Error("expected error, got nil")
	}
}
