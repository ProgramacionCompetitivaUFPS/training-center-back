package problem

import (
	"context"
	"strings"
	"testing"
	"time"

	domainProblem "github.com/training-judge-center/backend/internal/domain/problem"
	"github.com/training-judge-center/backend/internal/domain/shared"
)

// passingValidation is a check-only ticket that finds its problem passing.
func passingValidation(t *testing.T) *ValidateProblemUseCase {
	t.Helper()
	v, err := domainProblem.NewCheckOnlyValidation("validation-1", testProbID, shared.RestoreUserID(authorID), testNow)
	if err != nil {
		t.Fatalf("fixture: %v", err)
	}
	validationRepo := &mockValidationRepository{
		findByIDFn: func(_ context.Context, _ string) (*domainProblem.ProblemValidation, error) { return v, nil },
	}
	solutionValidator := &mockSolutionValidator{
		validateFn: func(_ context.Context, _ string) (*SolutionValidationResult, error) {
			return &SolutionValidationResult{SampleCases: 1, SecretCases: 2, SolutionsTested: 1, Passed: true}, nil
		},
	}
	return newValidateProblemUseCase(validationRepo, &mockJudgingPreparer{}, &mockJudgingArtifactWriter{}, solutionValidator, &mockProblemPublisher{})
}

func TestValidateProblem_LogsHowItEndedAndHowLongItWaited(t *testing.T) {
	logs := captureLogs(t)
	uc := passingValidation(t)

	in := ValidateProblemInput{ValidationID: "validation-1", EnqueuedAt: time.Now().Add(-3 * time.Second)}
	if err := uc.Execute(context.Background(), in); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := logs.String()
	if got := strings.Count(out, "validation finished"); got != 1 {
		t.Fatalf("logged %d lines, want exactly one per validation:\n%s", got, out)
	}
	for _, field := range []string{"validation_id=validation-1", "outcome=PASSED", "ran_for=", "queue_wait=3"} {
		if !strings.Contains(out, field) {
			t.Errorf("the line must carry %q:\n%s", field, out)
		}
	}
}

func TestValidateProblem_WithoutAnEnqueueTimeItLogsNoQueueWait(t *testing.T) {
	logs := captureLogs(t)
	uc := passingValidation(t)

	if err := uc.Execute(context.Background(), ValidateProblemInput{ValidationID: "validation-1"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if strings.Contains(logs.String(), "queue_wait") {
		t.Errorf("an unknown enqueue time must not be reported as a wait of billions of seconds:\n%s", logs)
	}
}

// A ticket that was not pending did not run, so a line saying it finished would be a lie.
func TestValidateProblem_ATicketThatIsNotPendingLogsNothing(t *testing.T) {
	logs := captureLogs(t)
	v, _ := domainProblem.NewCheckOnlyValidation("validation-1", testProbID, shared.RestoreUserID(authorID), testNow)
	_ = v.Start(testNow)
	validationRepo := &mockValidationRepository{
		findByIDFn: func(_ context.Context, _ string) (*domainProblem.ProblemValidation, error) { return v, nil },
	}
	uc := newValidateProblemUseCase(validationRepo, &mockJudgingPreparer{}, &mockJudgingArtifactWriter{}, &mockSolutionValidator{}, &mockProblemPublisher{})

	if err := uc.Execute(context.Background(), ValidateProblemInput{ValidationID: "validation-1"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if strings.Contains(logs.String(), "validation finished") {
		t.Errorf("a validation that did not run must not log that it finished:\n%s", logs)
	}
}
