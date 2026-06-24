package submission_test

import (
	"testing"
	"time"

	"github.com/training-judge-center/backend/internal/domain/shared"
	"github.com/training-judge-center/backend/internal/domain/submission"
)

var testNow = time.Date(2025, 1, 15, 12, 0, 0, 0, time.UTC)

const (
	testSubmissionID = "aaaaaaaa-0000-0000-0000-000000000001"
	testProblemID    = "bbbbbbbb-0000-0000-0000-000000000001"
	testUserID       = "cccccccc-0000-0000-0000-000000000001"
)

func newRunningSubmission(t *testing.T) *submission.Submission {
	t.Helper()
	lang := submission.RestoreLanguage("cpp20")
	s, err := submission.NewSubmission(testSubmissionID, testProblemID, shared.RestoreUserID(testUserID), nil, nil, lang, "g++", "gs://bucket/code.cpp", "abc123", 512, "", "", testNow)
	if err != nil {
		t.Fatalf("NewSubmission: %v", err)
	}
	if err := s.Start(testNow); err != nil {
		t.Fatalf("Start: %v", err)
	}
	return s
}

func TestNewSubmission_EmptyID_ReturnsError(t *testing.T) {
	lang := submission.RestoreLanguage("cpp20")
	_, err := submission.NewSubmission("", testProblemID, shared.RestoreUserID(testUserID), nil, nil, lang, "g++", "gs://bucket/code.cpp", "abc123", 512, "", "", testNow)
	if err == nil {
		t.Error("expected error for empty id, got nil")
	}
}

func TestNewSubmission_SetsInitialState(t *testing.T) {
	lang := submission.RestoreLanguage("cpp20")
	s, err := submission.NewSubmission(testSubmissionID, testProblemID, shared.RestoreUserID(testUserID), nil, nil, lang, "g++", "gs://bucket/code.cpp", "abc123", 512, "", "", testNow)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !s.Status().IsPending() {
		t.Errorf("expected PENDING, got %s", s.Status())
	}
	if s.JudgedAt() != nil {
		t.Error("expected nil JudgedAt")
	}
	if s.TimeMs() != nil {
		t.Error("expected nil TimeMs")
	}
	if s.MemoryKb() != nil {
		t.Error("expected nil MemoryKb")
	}
	if s.CompileLog() != nil {
		t.Error("expected nil CompileLog")
	}
	if got := s.SubmittedAt(); !got.Equal(testNow.UTC()) {
		t.Errorf("SubmittedAt: got %v, want %v", got, testNow.UTC())
	}
}

func TestSubmission_Start_FromPending_Succeeds(t *testing.T) {
	lang := submission.RestoreLanguage("cpp20")
	s, _ := submission.NewSubmission(testSubmissionID, testProblemID, shared.RestoreUserID(testUserID), nil, nil, lang, "g++", "path", "abc123", 512, "", "", testNow)

	if err := s.Start(testNow); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !s.Status().IsRunning() {
		t.Errorf("expected RUNNING, got %s", s.Status())
	}
}

func TestSubmission_Start_FromRunning_Fails(t *testing.T) {
	s := newRunningSubmission(t)
	if err := s.Start(testNow); err == nil {
		t.Error("expected error when Starting from RUNNING, got nil")
	}
}

func TestSubmission_MarkAccepted_FromRunning_Succeeds(t *testing.T) {
	s := newRunningSubmission(t)
	if err := s.MarkAccepted(150, 4096, testNow); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !s.Status().IsFinal() {
		t.Error("expected final status")
	}
	if s.Status().String() != "ACCEPTED" {
		t.Errorf("expected ACCEPTED, got %s", s.Status())
	}
	if s.TimeMs() == nil || *s.TimeMs() != 150 {
		t.Errorf("unexpected TimeMs: %v", s.TimeMs())
	}
	if s.MemoryKb() == nil || *s.MemoryKb() != 4096 {
		t.Errorf("unexpected MemoryKb: %v", s.MemoryKb())
	}
	if s.JudgedAt() == nil {
		t.Error("expected non-nil JudgedAt")
	}
}

func TestSubmission_MarkAccepted_FromPending_Fails(t *testing.T) {
	lang := submission.RestoreLanguage("cpp20")
	s, _ := submission.NewSubmission(testSubmissionID, testProblemID, shared.RestoreUserID(testUserID), nil, nil, lang, "g++", "path", "abc123", 512, "", "", testNow)
	if err := s.MarkAccepted(150, 4096, testNow); err == nil {
		t.Error("expected error when marking ACCEPTED from PENDING, got nil")
	}
}

func TestSubmission_MarkWrongAnswer(t *testing.T) {
	s := newRunningSubmission(t)
	if err := s.MarkWrongAnswer(200, 2048, testNow); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s.Status().String() != "WRONG_ANSWER" {
		t.Errorf("expected WRONG_ANSWER, got %s", s.Status())
	}
}

func TestSubmission_MarkTimeLimitExceeded(t *testing.T) {
	s := newRunningSubmission(t)
	if err := s.MarkTimeLimitExceeded(5000, testNow); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s.Status().String() != "TIME_LIMIT_EXCEEDED" {
		t.Errorf("expected TIME_LIMIT_EXCEEDED, got %s", s.Status())
	}
	if s.MemoryKb() != nil {
		t.Error("expected nil MemoryKb for TLE")
	}
}

func TestSubmission_MarkMemoryLimitExceeded(t *testing.T) {
	s := newRunningSubmission(t)
	if err := s.MarkMemoryLimitExceeded(262144, testNow); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s.Status().String() != "MEMORY_LIMIT_EXCEEDED" {
		t.Errorf("expected MEMORY_LIMIT_EXCEEDED, got %s", s.Status())
	}
	if s.TimeMs() != nil {
		t.Error("expected nil TimeMs for MLE")
	}
}

func TestSubmission_MarkRuntimeError(t *testing.T) {
	s := newRunningSubmission(t)
	if err := s.MarkRuntimeError(300, 1024, testNow); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s.Status().String() != "RUNTIME_ERROR" {
		t.Errorf("expected RUNTIME_ERROR, got %s", s.Status())
	}
}

func TestSubmission_MarkCompilationError(t *testing.T) {
	s := newRunningSubmission(t)
	if err := s.MarkCompilationError("error: undeclared identifier", testNow); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s.Status().String() != "COMPILATION_ERROR" {
		t.Errorf("expected COMPILATION_ERROR, got %s", s.Status())
	}
	if s.CompileLog() == nil || *s.CompileLog() != "error: undeclared identifier" {
		t.Errorf("unexpected CompileLog: %v", s.CompileLog())
	}
}

func TestSubmission_MarkSystemError(t *testing.T) {
	s := newRunningSubmission(t)
	if err := s.MarkSystemError(testNow); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s.Status().String() != "SYSTEM_ERROR" {
		t.Errorf("expected SYSTEM_ERROR, got %s", s.Status())
	}
}

func TestSubmission_MarkFinal_FromFinal_Fails(t *testing.T) {
	s := newRunningSubmission(t)
	_ = s.MarkAccepted(100, 512, testNow)
	if err := s.MarkWrongAnswer(100, 512, testNow); err == nil {
		t.Error("expected error when re-marking final submission, got nil")
	}
}
