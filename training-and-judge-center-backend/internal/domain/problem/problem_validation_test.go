package problem_test

import (
	"testing"
	"time"

	"github.com/training-judge-center/backend/internal/domain/problem"
	"github.com/training-judge-center/backend/internal/domain/shared"
)

func newPendingValidation(t *testing.T) *problem.ProblemValidation {
	t.Helper()
	v, err := problem.NewProblemValidation("validation-1", testProblemID, shared.RestoreUserID(testAuthorID), testNow)
	if err != nil {
		t.Fatalf("NewProblemValidation: unexpected error: %v", err)
	}
	return v
}

func TestNewProblemValidation_ValidInput_CreatesPendingValidation(t *testing.T) {
	v, err := problem.NewProblemValidation("validation-1", testProblemID, shared.RestoreUserID(testAuthorID), testNow)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v.ID() != "validation-1" {
		t.Errorf("ID(): got %q, want validation-1", v.ID())
	}
	if v.ProblemID() != testProblemID {
		t.Errorf("ProblemID(): got %q, want %q", v.ProblemID(), testProblemID)
	}
	if !v.Status().IsPending() {
		t.Error("Status(): expected IsPending() true for a freshly created validation")
	}
	if !v.RequestedAt().Equal(testNow.UTC()) {
		t.Errorf("RequestedAt(): got %v, want %v", v.RequestedAt(), testNow.UTC())
	}
	if v.CompletedAt() != nil {
		t.Error("CompletedAt(): expected nil for a freshly created validation")
	}
}

func TestNewProblemValidation_EmptyID_ReturnsError(t *testing.T) {
	_, err := problem.NewProblemValidation("", testProblemID, shared.RestoreUserID(testAuthorID), testNow)
	if err == nil {
		t.Error("expected error for empty id, got nil")
	}
}

func TestNewProblemValidation_EmptyProblemID_ReturnsError(t *testing.T) {
	_, err := problem.NewProblemValidation("validation-1", "", shared.RestoreUserID(testAuthorID), testNow)
	if err == nil {
		t.Error("expected error for empty problemID, got nil")
	}
}

func TestRestoreProblemValidation_PreservesFields(t *testing.T) {
	completedAt := testNow.Add(time.Minute)
	result := `{"validationLogs":["ok"]}`
	status := problem.RestoreProblemValidationStatus("PASSED")

	v := problem.RestoreProblemValidation(
		"validation-1", testProblemID, shared.RestoreUserID(testAuthorID),
		status, testNow, &completedAt, &result, nil,
	)

	if v.ID() != "validation-1" {
		t.Errorf("ID(): got %q, want validation-1", v.ID())
	}
	if !v.Status().IsPassed() {
		t.Error("Status(): expected IsPassed() true")
	}
	if v.CompletedAt() == nil || !v.CompletedAt().Equal(completedAt) {
		t.Errorf("CompletedAt(): got %v, want %v", v.CompletedAt(), completedAt)
	}
	if v.ResultJSON() == nil || *v.ResultJSON() != result {
		t.Errorf("ResultJSON(): got %v, want %q", v.ResultJSON(), result)
	}
}

func TestStart_Pending_TransitionsToRunning(t *testing.T) {
	v := newPendingValidation(t)

	if err := v.Start(testNow); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !v.Status().IsRunning() {
		t.Error("Status(): expected IsRunning() true after Start")
	}
}

func TestStart_AlreadyRunning_ReturnsError(t *testing.T) {
	v := newPendingValidation(t)
	_ = v.Start(testNow)

	if err := v.Start(testNow); err == nil {
		t.Error("expected error when starting an already-running validation, got nil")
	}
}

func TestMarkPassed_Running_TransitionsToPassed(t *testing.T) {
	v := newPendingValidation(t)
	_ = v.Start(testNow)
	later := testNow.Add(time.Minute)

	if err := v.MarkPassed(`{"validationLogs":["ok"]}`, later); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !v.Status().IsPassed() {
		t.Error("Status(): expected IsPassed() true")
	}
	if v.CompletedAt() == nil || !v.CompletedAt().Equal(later.UTC()) {
		t.Errorf("CompletedAt(): got %v, want %v", v.CompletedAt(), later.UTC())
	}
}

func TestMarkPassed_NotRunning_ReturnsError(t *testing.T) {
	v := newPendingValidation(t)

	if err := v.MarkPassed(`{}`, testNow); err == nil {
		t.Error("expected error when marking passed without starting, got nil")
	}
}

func TestMarkFailed_Running_TransitionsToFailed(t *testing.T) {
	v := newPendingValidation(t)
	_ = v.Start(testNow)

	if err := v.MarkFailed(`{"missingFields":["statement"]}`, testNow); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !v.Status().IsFinal() {
		t.Error("Status(): expected IsFinal() true")
	}
	if v.Status().IsPassed() {
		t.Error("Status(): expected IsPassed() false for a failed validation")
	}
}

func TestMarkSystemError_Running_TransitionsToSystemError(t *testing.T) {
	v := newPendingValidation(t)
	_ = v.Start(testNow)

	if err := v.MarkSystemError(`{"validationLogs":["internal error"]}`, testNow); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !v.Status().IsSystemError() {
		t.Error("Status(): expected IsSystemError() true")
	}
}

func TestNewPendingJudgingChange_NeedsSomethingToChange(t *testing.T) {
	if _, err := problem.NewPendingJudgingChange(nil, nil); err == nil {
		t.Error("expected an error for a change that replaces nothing")
	}

	key := "problems/p/testcases/x"
	if _, err := problem.NewPendingJudgingChange(&key, nil); err != nil {
		t.Errorf("test cases alone should be a valid change, got %v", err)
	}
}

func TestNewJudgingUpdateValidation_CarriesTheChange(t *testing.T) {
	key := "problems/p/testcases/x"
	change, err := problem.NewPendingJudgingChange(&key, nil)
	if err != nil {
		t.Fatalf("NewPendingJudgingChange: %v", err)
	}

	v, err := problem.NewJudgingUpdateValidation("validation-1", testProblemID, shared.RestoreUserID(testAuthorID), change, testNow)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !v.IsJudgingUpdate() || v.PendingChange() == nil || *v.PendingChange().TestCasesKey() != key {
		t.Errorf("PendingChange(): got %+v", v.PendingChange())
	}
	if !v.Status().IsPending() {
		t.Error("a new judging update starts PENDING")
	}
	if newPendingValidation(t).IsJudgingUpdate() {
		t.Error("a publish validation is not a judging update")
	}
}
