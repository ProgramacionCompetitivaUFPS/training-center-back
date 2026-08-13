package judge

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/training-judge-center/backend/internal/domain/submission"
)

func newValidateSolutionsUseCase(
	solutions *mockSolutionProvider,
	downloader *mockSourceCodeDownloader,
	problems *mockProblemProvider,
	testCases *mockTestCaseProvider,
	executor *mockExecutor,
	checker *mockOutputChecker,
) *ValidateSolutionsUseCase {
	uc := NewValidateSolutionsUseCase(
		solutions, downloader, problems, testCases,
		executor, checker,
		RetryConfig{MaxAttempts: 1, BackoffBase: 0},
	)
	uc.sleep = func(time.Duration) {}
	return uc
}

func TestValidateSolutions_AllPass_ReturnsPassed(t *testing.T) {
	uc := newValidateSolutionsUseCase(&mockSolutionProvider{}, &mockSourceCodeDownloader{}, &mockProblemProvider{}, &mockTestCaseProvider{}, &mockExecutor{}, &mockOutputChecker{})

	out, err := uc.Execute(context.Background(), ValidateSolutionsInput{ProblemID: problemID})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !out.Passed {
		t.Error("expected Passed=true")
	}
	if out.SolutionsTested != 1 {
		t.Errorf("SolutionsTested: got %d, want 1", out.SolutionsTested)
	}
	if out.Failure != nil {
		t.Errorf("expected no Failure, got %+v", out.Failure)
	}
}

func TestValidateSolutions_NoSolutions_ReturnsInternalError(t *testing.T) {
	solutions := &mockSolutionProvider{
		getSolutionsFn: func(_ context.Context, _ string) ([]Solution, error) {
			return []Solution{}, nil
		},
	}
	uc := newValidateSolutionsUseCase(solutions, &mockSourceCodeDownloader{}, &mockProblemProvider{}, &mockTestCaseProvider{}, &mockExecutor{}, &mockOutputChecker{})

	_, err := uc.Execute(context.Background(), ValidateSolutionsInput{ProblemID: problemID})
	if err == nil {
		t.Fatal("expected an error when there are no solutions")
	}
}

func TestValidateSolutions_CompileFails_ReturnsCompileErrorFailure(t *testing.T) {
	executor := &mockExecutor{
		beginSessionFn: func(_ context.Context, _ submission.Language) (ExecutionSession, error) {
			return &mockExecutionSession{
				compileFn: func(_ context.Context, _ CompileRequest) (CompileResult, error) {
					return CompileResult{Success: false, Log: "syntax error"}, nil
				},
			}, nil
		},
	}
	uc := newValidateSolutionsUseCase(&mockSolutionProvider{}, &mockSourceCodeDownloader{}, &mockProblemProvider{}, &mockTestCaseProvider{}, executor, &mockOutputChecker{})

	out, err := uc.Execute(context.Background(), ValidateSolutionsInput{ProblemID: problemID})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Passed {
		t.Fatal("expected Passed=false")
	}
	if out.Failure.Kind != FailureCompileError || out.Failure.CompileLog != "syntax error" {
		t.Errorf("Failure: got %+v", out.Failure)
	}
}

func TestValidateSolutions_WrongAnswer_ReturnsFailure(t *testing.T) {
	testCases := &mockTestCaseProvider{
		getTestCasesFn: func(_ context.Context, _ string) ([]TestCase, error) {
			return []TestCase{{Name: "secret/001", Input: []byte("1 2"), ExpectedOutput: []byte("5")}}, nil
		},
	}
	checker := &mockOutputChecker{
		checkFn: func(_ context.Context, _ CheckRequest) (CheckResult, error) {
			return CheckResult{Accepted: false}, nil
		},
	}
	uc := newValidateSolutionsUseCase(&mockSolutionProvider{}, &mockSourceCodeDownloader{}, &mockProblemProvider{}, testCases, &mockExecutor{}, checker)

	out, err := uc.Execute(context.Background(), ValidateSolutionsInput{ProblemID: problemID})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Failure == nil || out.Failure.Kind != FailureWrongAnswer || out.Failure.TestCase != "secret/001" {
		t.Errorf("Failure: got %+v", out.Failure)
	}
	if string(out.Failure.Expected) != "5" || string(out.Failure.Actual) != "3" {
		t.Errorf("Expected/Actual: got %q/%q, want %q/%q", out.Failure.Expected, out.Failure.Actual, "5", "3")
	}
}

func TestValidateSolutions_TimeLimitExceeded_ExitCode124(t *testing.T) {
	testCases := &mockTestCaseProvider{
		getTestCasesFn: func(_ context.Context, _ string) ([]TestCase, error) {
			return []TestCase{{Name: "secret/001", Input: []byte("1 2")}}, nil
		},
	}
	executor := &mockExecutor{
		beginSessionFn: func(_ context.Context, _ submission.Language) (ExecutionSession, error) {
			return &mockExecutionSession{
				runTestCaseFn: func(_ context.Context, _ RunRequest) (RunResult, error) {
					return RunResult{ExitCode: exitCodeTLE, TimeMs: 1000}, nil
				},
			}, nil
		},
	}
	uc := newValidateSolutionsUseCase(&mockSolutionProvider{}, &mockSourceCodeDownloader{}, &mockProblemProvider{}, testCases, executor, &mockOutputChecker{})

	out, err := uc.Execute(context.Background(), ValidateSolutionsInput{ProblemID: problemID})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Failure == nil || out.Failure.Kind != FailureTimeLimitExceeded || out.Failure.TimeMs != 1000 {
		t.Errorf("Failure: got %+v", out.Failure)
	}
	if out.Failure.TimeLimitMs != 1000 {
		t.Errorf("TimeLimitMs: got %d, want 1000 (from the default ProblemLimits mock)", out.Failure.TimeLimitMs)
	}
}

func TestValidateSolutions_TimeLimitExceeded_OverLimitDespiteExitZero(t *testing.T) {
	problems := &mockProblemProvider{
		getLimitsFn: func(_ context.Context, _ string) (ProblemLimits, error) {
			return ProblemLimits{TimeLimitMs: 500, MemoryKb: 262144}, nil
		},
	}
	testCases := &mockTestCaseProvider{
		getTestCasesFn: func(_ context.Context, _ string) ([]TestCase, error) {
			return []TestCase{{Name: "secret/001", Input: []byte("1 2")}}, nil
		},
	}
	executor := &mockExecutor{
		beginSessionFn: func(_ context.Context, _ submission.Language) (ExecutionSession, error) {
			return &mockExecutionSession{
				runTestCaseFn: func(_ context.Context, _ RunRequest) (RunResult, error) {
					return RunResult{ExitCode: 0, TimeMs: 600}, nil
				},
			}, nil
		},
	}
	uc := newValidateSolutionsUseCase(&mockSolutionProvider{}, &mockSourceCodeDownloader{}, problems, testCases, executor, &mockOutputChecker{})

	out, err := uc.Execute(context.Background(), ValidateSolutionsInput{ProblemID: problemID})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Failure == nil || out.Failure.Kind != FailureTimeLimitExceeded {
		t.Errorf("Failure: got %+v", out.Failure)
	}
}

func TestValidateSolutions_MemoryLimitExceeded_ExitCode137(t *testing.T) {
	testCases := &mockTestCaseProvider{
		getTestCasesFn: func(_ context.Context, _ string) ([]TestCase, error) {
			return []TestCase{{Name: "secret/001", Input: []byte("1 2")}}, nil
		},
	}
	executor := &mockExecutor{
		beginSessionFn: func(_ context.Context, _ submission.Language) (ExecutionSession, error) {
			return &mockExecutionSession{
				runTestCaseFn: func(_ context.Context, _ RunRequest) (RunResult, error) {
					return RunResult{ExitCode: exitCodeMLE}, nil
				},
			}, nil
		},
	}
	uc := newValidateSolutionsUseCase(&mockSolutionProvider{}, &mockSourceCodeDownloader{}, &mockProblemProvider{}, testCases, executor, &mockOutputChecker{})

	out, err := uc.Execute(context.Background(), ValidateSolutionsInput{ProblemID: problemID})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Failure == nil || out.Failure.Kind != FailureMemoryLimitExceeded {
		t.Errorf("Failure: got %+v", out.Failure)
	}
}

func TestValidateSolutions_RuntimeError_UnknownExitCode(t *testing.T) {
	testCases := &mockTestCaseProvider{
		getTestCasesFn: func(_ context.Context, _ string) ([]TestCase, error) {
			return []TestCase{{Name: "secret/001", Input: []byte("1 2")}}, nil
		},
	}
	executor := &mockExecutor{
		beginSessionFn: func(_ context.Context, _ submission.Language) (ExecutionSession, error) {
			return &mockExecutionSession{
				runTestCaseFn: func(_ context.Context, _ RunRequest) (RunResult, error) {
					return RunResult{ExitCode: 1}, nil
				},
			}, nil
		},
	}
	uc := newValidateSolutionsUseCase(&mockSolutionProvider{}, &mockSourceCodeDownloader{}, &mockProblemProvider{}, testCases, executor, &mockOutputChecker{})

	out, err := uc.Execute(context.Background(), ValidateSolutionsInput{ProblemID: problemID})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Failure == nil || out.Failure.Kind != FailureRuntimeError {
		t.Errorf("Failure: got %+v", out.Failure)
	}
}

func TestValidateSolutions_StopsAtFirstFailingSolution(t *testing.T) {
	solutions := &mockSolutionProvider{
		getSolutionsFn: func(_ context.Context, _ string) ([]Solution, error) {
			return []Solution{
				{FileKey: "sol1.cpp", Language: submission.RestoreLanguage("cpp20")},
				{FileKey: "sol2.cpp", Language: submission.RestoreLanguage("cpp20")},
			}, nil
		},
	}
	executor := &mockExecutor{
		beginSessionFn: func(_ context.Context, _ submission.Language) (ExecutionSession, error) {
			return &mockExecutionSession{
				compileFn: func(_ context.Context, _ CompileRequest) (CompileResult, error) {
					return CompileResult{Success: false, Log: "syntax error"}, nil
				},
			}, nil
		},
	}
	uc := newValidateSolutionsUseCase(solutions, &mockSourceCodeDownloader{}, &mockProblemProvider{}, &mockTestCaseProvider{}, executor, &mockOutputChecker{})

	out, err := uc.Execute(context.Background(), ValidateSolutionsInput{ProblemID: problemID})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.SolutionsTested != 1 {
		t.Errorf("SolutionsTested: got %d, want 1", out.SolutionsTested)
	}
	if executor.calls != 1 {
		t.Errorf("expected BeginSession called once (stopped at first failure), got %d", executor.calls)
	}
	if out.Failure.FileKey != "sol1.cpp" {
		t.Errorf("Failure.FileKey: got %q, want sol1.cpp", out.Failure.FileKey)
	}
}

func TestValidateSolutions_CountsSampleAndSecretCases(t *testing.T) {
	testCases := &mockTestCaseProvider{
		getTestCasesFn: func(_ context.Context, _ string) ([]TestCase, error) {
			return []TestCase{
				{Name: "sample/001", Input: []byte("1"), ExpectedOutput: []byte("1")},
				{Name: "sample/002", Input: []byte("2"), ExpectedOutput: []byte("2")},
				{Name: "secret/001", Input: []byte("3"), ExpectedOutput: []byte("3")},
			}, nil
		},
	}
	uc := newValidateSolutionsUseCase(&mockSolutionProvider{}, &mockSourceCodeDownloader{}, &mockProblemProvider{}, testCases, &mockExecutor{}, &mockOutputChecker{})

	out, err := uc.Execute(context.Background(), ValidateSolutionsInput{ProblemID: problemID})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.SampleCases != 2 || out.SecretCases != 1 {
		t.Errorf("SampleCases/SecretCases: got %d/%d, want 2/1", out.SampleCases, out.SecretCases)
	}
}

func TestValidateSolutions_TransientError_RetriesAndSucceeds(t *testing.T) {
	executor := &mockExecutor{
		beginSessionFn: func(_ context.Context, _ submission.Language) (ExecutionSession, error) {
			return nil, errTransient
		},
	}
	uc := newValidateSolutionsUseCase(&mockSolutionProvider{}, &mockSourceCodeDownloader{}, &mockProblemProvider{}, &mockTestCaseProvider{}, executor, &mockOutputChecker{})
	uc.retry = RetryConfig{MaxAttempts: 3, BackoffBase: 0}

	// Succeed on the 3rd BeginSession call.
	attempt := 0
	executor.beginSessionFn = func(_ context.Context, _ submission.Language) (ExecutionSession, error) {
		attempt++
		if attempt < 3 {
			return nil, errTransient
		}
		return &mockExecutionSession{}, nil
	}

	out, err := uc.Execute(context.Background(), ValidateSolutionsInput{ProblemID: problemID})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !out.Passed {
		t.Error("expected Passed=true after retries succeed")
	}
	if executor.calls != 3 {
		t.Errorf("expected BeginSession called 3 times, got %d", executor.calls)
	}
}

func TestValidateSolutions_TransientError_ExhaustsRetries_ReturnsError(t *testing.T) {
	executor := &mockExecutor{
		beginSessionFn: func(_ context.Context, _ submission.Language) (ExecutionSession, error) {
			return nil, errors.New("docker unavailable")
		},
	}
	uc := newValidateSolutionsUseCase(&mockSolutionProvider{}, &mockSourceCodeDownloader{}, &mockProblemProvider{}, &mockTestCaseProvider{}, executor, &mockOutputChecker{})
	uc.retry = RetryConfig{MaxAttempts: 3, BackoffBase: 0}

	_, err := uc.Execute(context.Background(), ValidateSolutionsInput{ProblemID: problemID})
	if err == nil {
		t.Fatal("expected an error after exhausting retries")
	}
	if executor.calls != 3 {
		t.Errorf("expected BeginSession called 3 times, got %d", executor.calls)
	}
}

func TestTruncatePreview_ShortData_Unchanged(t *testing.T) {
	data := []byte("short output")
	got := truncatePreview(data)
	if string(got) != "short output" {
		t.Errorf("got %q, want unchanged input", got)
	}
}

func TestTruncatePreview_LongData_TruncatedWithSuffix(t *testing.T) {
	data := make([]byte, maxFailurePreviewBytes+500)
	for i := range data {
		data[i] = 'a'
	}

	got := truncatePreview(data)
	if len(got) != maxFailurePreviewBytes+len("... (truncated)") {
		t.Errorf("length: got %d, want %d", len(got), maxFailurePreviewBytes+len("... (truncated)"))
	}
	if string(got[maxFailurePreviewBytes:]) != "... (truncated)" {
		t.Errorf("suffix: got %q", got[maxFailurePreviewBytes:])
	}
}

func TestValidateSolutions_WrongAnswer_LargeOutput_IsTruncated(t *testing.T) {
	largeOutput := make([]byte, maxFailurePreviewBytes+1000)
	for i := range largeOutput {
		largeOutput[i] = 'x'
	}

	testCases := &mockTestCaseProvider{
		getTestCasesFn: func(_ context.Context, _ string) ([]TestCase, error) {
			return []TestCase{{Name: "secret/001", Input: []byte("1"), ExpectedOutput: largeOutput}}, nil
		},
	}
	executor := &mockExecutor{
		beginSessionFn: func(_ context.Context, _ submission.Language) (ExecutionSession, error) {
			return &mockExecutionSession{
				runTestCaseFn: func(_ context.Context, _ RunRequest) (RunResult, error) {
					return RunResult{ExitCode: 0, Output: largeOutput}, nil
				},
			}, nil
		},
	}
	checker := &mockOutputChecker{
		checkFn: func(_ context.Context, _ CheckRequest) (CheckResult, error) {
			return CheckResult{Accepted: false}, nil
		},
	}
	uc := newValidateSolutionsUseCase(&mockSolutionProvider{}, &mockSourceCodeDownloader{}, &mockProblemProvider{}, testCases, executor, checker)

	out, err := uc.Execute(context.Background(), ValidateSolutionsInput{ProblemID: problemID})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out.Failure.Expected) != maxFailurePreviewBytes+len("... (truncated)") {
		t.Errorf("Expected length: got %d, want the truncated length", len(out.Failure.Expected))
	}
	if len(out.Failure.Actual) != maxFailurePreviewBytes+len("... (truncated)") {
		t.Errorf("Actual length: got %d, want the truncated length", len(out.Failure.Actual))
	}
}
