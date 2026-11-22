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
		status, testNow, &completedAt, &result, nil, true,
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

func TestNewProblemValidation_PublishesOnPassByDefault(t *testing.T) {
	v := newPendingValidation(t)

	if !v.PublishesOnPass() {
		t.Error("PublishesOnPass(): a regular ticket must publish the problem when it passes")
	}
	if v.RequestedBy().Value() != testAuthorID {
		t.Errorf("RequestedBy(): got %q, want %q", v.RequestedBy().Value(), testAuthorID)
	}
	if v.IsJudgingUpdate() {
		t.Error("IsJudgingUpdate(): a regular ticket carries no pending change")
	}
}

func TestNewCheckOnlyValidation_DoesNotPublishOnPass(t *testing.T) {
	v, err := problem.NewCheckOnlyValidation("validation-1", testProblemID, shared.RestoreUserID(testAuthorID), testNow)

	if err != nil {
		t.Fatalf("NewCheckOnlyValidation: %v", err)
	}
	if v.PublishesOnPass() {
		t.Error("PublishesOnPass(): a check-only ticket must leave the problem unpublished")
	}
	if !v.Status().IsPending() {
		t.Error("Status(): a check-only ticket starts pending")
	}
}

func TestNewCheckOnlyValidation_MissingIdentifiers_IsAnInternalError(t *testing.T) {
	tests := []struct {
		name        string
		id, problem string
	}{
		{"an empty id", "", testProblemID},
		{"an empty problem id", "validation-1", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v, err := problem.NewCheckOnlyValidation(tt.id, tt.problem, shared.RestoreUserID(testAuthorID), testNow)

			assertInternalError(t, tt.name, err)
			if v != nil {
				t.Errorf("expected no validation, got %+v", v)
			}
		})
	}
}

func TestMarkFinal_NotRunning_IsAnInternalErrorAndChangesNothing(t *testing.T) {
	marks := map[string]func(v *problem.ProblemValidation) error{
		"passed":       func(v *problem.ProblemValidation) error { return v.MarkPassed(`{}`, testNow) },
		"failed":       func(v *problem.ProblemValidation) error { return v.MarkFailed(`{}`, testNow) },
		"system error": func(v *problem.ProblemValidation) error { return v.MarkSystemError(`{}`, testNow) },
	}

	for name, mark := range marks {
		t.Run(name+" while pending", func(t *testing.T) {
			v := newPendingValidation(t)

			err := mark(v)

			assertInternalError(t, name, err)
			if !v.Status().IsPending() || v.CompletedAt() != nil || v.ResultJSON() != nil {
				t.Errorf("a rejected transition changed the ticket: status pending=%v, completedAt=%v, result=%v", v.Status().IsPending(), v.CompletedAt(), v.ResultJSON())
			}
		})
		t.Run(name+" after it already finished", func(t *testing.T) {
			v := newPendingValidation(t)
			_ = v.Start(testNow)
			_ = v.MarkPassed(`{"first":true}`, testNow)

			err := mark(v)

			assertInternalError(t, name, err)
			if !v.Status().IsPassed() || *v.ResultJSON() != `{"first":true}` {
				t.Errorf("a finished ticket was changed: passed=%v, result=%v", v.Status().IsPassed(), *v.ResultJSON())
			}
		})
	}
}

func TestMarkFinal_Running_RecordsTheResultAndTheInstantInUTC(t *testing.T) {
	v := newPendingValidation(t)
	_ = v.Start(testNow)
	localNow := time.Date(2024, 1, 15, 14, 0, 0, 0, time.FixedZone("UTC+2", 2*3600))

	if err := v.MarkFailed(`{"missingFields":["statement"]}`, localNow); err != nil {
		t.Fatalf("MarkFailed: %v", err)
	}

	if v.ResultJSON() == nil || *v.ResultJSON() != `{"missingFields":["statement"]}` {
		t.Errorf("ResultJSON() = %v, want the stored result", v.ResultJSON())
	}
	if want := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC); v.CompletedAt() == nil || !v.CompletedAt().Equal(want) || v.CompletedAt().Location() != time.UTC {
		t.Errorf("CompletedAt() = %v, want %v in UTC", v.CompletedAt(), want)
	}
}

func TestPendingJudgingChange_ExposesTheCheckerAndRestoreKeepsEmptyChanges(t *testing.T) {
	checker := problem.RestoreJudgingFile("checker.cpp", "problems/p/checker.cpp", "cpp20", nil, nil)

	change, err := problem.NewPendingJudgingChange(nil, &checker)

	if err != nil {
		t.Fatalf("NewPendingJudgingChange: %v", err)
	}
	if change.Checker() == nil || change.Checker().Filename() != "checker.cpp" || change.TestCasesKey() != nil {
		t.Errorf("change = checker %+v, test cases %v, want only the checker", change.Checker(), change.TestCasesKey())
	}
	empty := problem.RestorePendingJudgingChange(nil, nil)
	if empty.Checker() != nil || empty.TestCasesKey() != nil {
		t.Errorf("a restored empty change must be kept as stored, got %+v/%v", empty.Checker(), empty.TestCasesKey())
	}
}

func TestNewJudgingUpdateValidation_MissingIdentifiers_IsAnInternalError(t *testing.T) {
	key := "problems/p/testcases/x"
	change, err := problem.NewPendingJudgingChange(&key, nil)
	if err != nil {
		t.Fatalf("NewPendingJudgingChange: %v", err)
	}

	v, err := problem.NewJudgingUpdateValidation("", testProblemID, shared.RestoreUserID(testAuthorID), change, testNow)

	assertInternalError(t, "empty id", err)
	if v != nil {
		t.Errorf("expected no validation, got %+v", v)
	}
}
