package problem

import (
	"context"
	"errors"
	"testing"

	appJudge "github.com/training-judge-center/backend/internal/application/judge"
	appProblem "github.com/training-judge-center/backend/internal/application/problem"
	"github.com/training-judge-center/backend/internal/domain/submission"
	"github.com/training-judge-center/backend/pkg/apperror"
)

// These mocks satisfy application/judge's exported ports just enough to
// drive ValidateSolutionsUseCase through the adapter under test — they don't
// re-test judge's own execution logic (validate_solutions_test.go already
// does that thoroughly), only that this adapter translates its result.

type mockJudgeSolutionProvider struct{}

func (m *mockJudgeSolutionProvider) GetSolutions(_ context.Context, _ string) ([]appJudge.Solution, error) {
	return []appJudge.Solution{{FileKey: "sol.cpp", Language: submission.RestoreLanguage("cpp20")}}, nil
}

type mockJudgeSourceCodeDownloader struct{}

func (m *mockJudgeSourceCodeDownloader) Download(_ context.Context, _ string) ([]byte, error) {
	return []byte("int main(){}"), nil
}

type mockJudgeProblemProvider struct{}

func (m *mockJudgeProblemProvider) GetLimits(_ context.Context, _ string) (appJudge.ProblemLimits, error) {
	return appJudge.ProblemLimits{TimeLimitMs: 1000, MemoryKb: 262144}, nil
}

type mockJudgeTestCaseProvider struct{}

func (m *mockJudgeTestCaseProvider) GetTestCases(_ context.Context, _ string) ([]appJudge.TestCase, error) {
	return []appJudge.TestCase{{Name: "sample/001", Input: []byte("1"), ExpectedOutput: []byte("1")}}, nil
}

type mockJudgeExecutionSession struct {
	compileFn func(ctx context.Context, req appJudge.CompileRequest) (appJudge.CompileResult, error)
}

func (m *mockJudgeExecutionSession) Compile(ctx context.Context, req appJudge.CompileRequest) (appJudge.CompileResult, error) {
	if m.compileFn != nil {
		return m.compileFn(ctx, req)
	}
	return appJudge.CompileResult{Success: true}, nil
}

func (m *mockJudgeExecutionSession) RunTestCase(_ context.Context, _ appJudge.RunRequest) (appJudge.RunResult, error) {
	return appJudge.RunResult{ExitCode: 0, TimeMs: 10, OutputPreview: []byte("1")}, nil
}

func (m *mockJudgeExecutionSession) Close(_ context.Context) error { return nil }

type mockJudgeExecutor struct {
	session *mockJudgeExecutionSession
}

func (m *mockJudgeExecutor) BeginSession(_ context.Context, _ submission.Language, _ int, _ string) (appJudge.ExecutionSession, error) {
	if m.session != nil {
		return m.session, nil
	}
	return &mockJudgeExecutionSession{}, nil
}

type mockJudgeOutputChecker struct{}

func (m *mockJudgeOutputChecker) BeginChecking(_ context.Context, _ string, _ submission.Language, _ string) (appJudge.CheckerSession, error) {
	return &mockJudgeCheckerSession{}, nil
}

type mockJudgeCheckerSession struct{}

func (m *mockJudgeCheckerSession) Check(_ context.Context, _ []byte) (appJudge.CheckResult, error) {
	return appJudge.CheckResult{Accepted: true}, nil
}

func (m *mockJudgeCheckerSession) Close(context.Context) error { return nil }

func newTestSolutionValidator(
	solutions appJudge.SolutionProvider,
	problems appJudge.ProblemProvider,
	testCases appJudge.TestCaseProvider,
	executor appJudge.Executor,
) *SolutionValidator {
	uc := appJudge.NewValidateSolutionsUseCase(
		solutions,
		&mockJudgeSourceCodeDownloader{},
		problems,
		testCases,
		executor,
		&mockJudgeOutputChecker{},
		appJudge.RetryConfig{MaxAttempts: 1},
	)
	return NewSolutionValidator(uc)
}

func TestSolutionValidator_AllPass_TranslatesToPassedResult(t *testing.T) {
	v := newTestSolutionValidator(&mockJudgeSolutionProvider{}, &mockJudgeProblemProvider{}, &mockJudgeTestCaseProvider{}, &mockJudgeExecutor{})

	result, err := v.Validate(context.Background(), testValidationProblemID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Passed {
		t.Error("expected Passed=true")
	}
	if result.SolutionsTested != 1 {
		t.Errorf("SolutionsTested: got %d, want 1", result.SolutionsTested)
	}
	if result.SampleCases != 1 {
		t.Errorf("SampleCases: got %d, want 1", result.SampleCases)
	}
	if result.Failure != nil {
		t.Errorf("expected no Failure, got %+v", result.Failure)
	}
}

func TestSolutionValidator_CompileFailure_TranslatesFailureFields(t *testing.T) {
	executor := &mockJudgeExecutor{
		session: &mockJudgeExecutionSession{
			compileFn: func(_ context.Context, _ appJudge.CompileRequest) (appJudge.CompileResult, error) {
				return appJudge.CompileResult{Success: false, Log: "syntax error"}, nil
			},
		},
	}
	v := newTestSolutionValidator(&mockJudgeSolutionProvider{}, &mockJudgeProblemProvider{}, &mockJudgeTestCaseProvider{}, executor)

	result, err := v.Validate(context.Background(), testValidationProblemID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Passed {
		t.Fatal("expected Passed=false")
	}
	if result.Failure == nil {
		t.Fatal("expected a Failure")
	}
	if result.Failure.Kind != appProblem.SolutionFailureCompileError {
		t.Errorf("Kind: got %q, want %q", result.Failure.Kind, appProblem.SolutionFailureCompileError)
	}
	if result.Failure.CompileLog != "syntax error" {
		t.Errorf("CompileLog: got %q, want %q", result.Failure.CompileLog, "syntax error")
	}
	if result.Failure.FileKey != "sol.cpp" {
		t.Errorf("FileKey: got %q, want %q", result.Failure.FileKey, "sol.cpp")
	}
}

func TestSolutionValidator_UnderlyingError_Propagates(t *testing.T) {
	wantErr := apperror.NewNotFound("PROBLEM_NOT_FOUND", "problem not found")
	solutions := &erroringSolutionProvider{err: wantErr}
	v := newTestSolutionValidator(solutions, &mockJudgeProblemProvider{}, &mockJudgeTestCaseProvider{}, &mockJudgeExecutor{})

	_, err := v.Validate(context.Background(), testValidationProblemID)
	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Code != "PROBLEM_NOT_FOUND" {
		t.Errorf("expected the underlying error to propagate untouched, got %v", err)
	}
}

type erroringSolutionProvider struct{ err error }

func (m *erroringSolutionProvider) GetSolutions(_ context.Context, _ string) ([]appJudge.Solution, error) {
	return nil, m.err
}
