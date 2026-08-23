package judge

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/training-judge-center/backend/internal/domain/submission"
)

func newJudgeSubmissionUseCase(
	updater *mockSubmissionUpdater,
	downloader *mockSourceCodeDownloader,
	problems *mockProblemProvider,
	testCases *mockTestCaseProvider,
	executor *mockExecutor,
	checker *mockOutputChecker,
) *JudgeSubmissionUseCase {
	uc := NewJudgeSubmissionUseCase(
		updater, downloader, problems, testCases,
		executor, checker, &mockTransactionManager{},
		RetryConfig{MaxAttempts: 1, BackoffBase: 0},
	)
	uc.sleep = func(time.Duration) {}
	return uc
}

func TestJudgeSubmission_NotPending_IsIgnored(t *testing.T) {
	runningSub := pendingSubmission()
	_ = runningSub.Start(testNow)

	executorCalled := false
	uc := newJudgeSubmissionUseCase(
		&mockSubmissionUpdater{
			getByIDFn: func(_ context.Context, _ submission.SubmissionID) (*submission.Submission, error) {
				return runningSub, nil
			},
		},
		&mockSourceCodeDownloader{},
		&mockProblemProvider{},
		&mockTestCaseProvider{},
		&mockExecutor{
			beginSessionFn: func(_ context.Context, _ submission.Language, _ int, _ string) (ExecutionSession, error) {
				executorCalled = true
				return &mockExecutionSession{}, nil
			},
		},
		&mockOutputChecker{},
	)

	if err := uc.Execute(context.Background(), JudgeSubmissionInput{SubmissionID: submissionID}); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if executorCalled {
		t.Error("executor should not be called for non-pending submission")
	}
}

func TestJudgeSubmission_SourceCodeDownloadError_MarksSystemError(t *testing.T) {
	var updatedStatus string
	uc := newJudgeSubmissionUseCase(
		&mockSubmissionUpdater{
			updateFn: func(_ context.Context, s *submission.Submission) error {
				updatedStatus = s.Status().String()
				return nil
			},
		},
		&mockSourceCodeDownloader{
			downloadFn: func(_ context.Context, _ string) ([]byte, error) {
				return nil, errTransient
			},
		},
		&mockProblemProvider{},
		&mockTestCaseProvider{},
		&mockExecutor{},
		&mockOutputChecker{},
	)

	if err := uc.Execute(context.Background(), JudgeSubmissionInput{SubmissionID: submissionID}); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if updatedStatus != "SYSTEM_ERROR" {
		t.Errorf("expected SYSTEM_ERROR, got %s", updatedStatus)
	}
}

func TestJudgeSubmission_CompilationError_MarksSubAndACKs(t *testing.T) {
	var updatedStatus string
	uc := newJudgeSubmissionUseCase(
		&mockSubmissionUpdater{
			updateFn: func(_ context.Context, s *submission.Submission) error {
				updatedStatus = s.Status().String()
				return nil
			},
		},
		&mockSourceCodeDownloader{},
		&mockProblemProvider{},
		&mockTestCaseProvider{},
		&mockExecutor{
			beginSessionFn: func(_ context.Context, _ submission.Language, _ int, _ string) (ExecutionSession, error) {
				return &mockExecutionSession{
					compileFn: func(_ context.Context, _ CompileRequest) (CompileResult, error) {
						return CompileResult{Success: false, Log: "error: undeclared"}, nil
					},
				}, nil
			},
		},
		&mockOutputChecker{},
	)

	if err := uc.Execute(context.Background(), JudgeSubmissionInput{SubmissionID: submissionID}); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if updatedStatus != "COMPILATION_ERROR" {
		t.Errorf("expected COMPILATION_ERROR, got %s", updatedStatus)
	}
}

func TestJudgeSubmission_Accepted(t *testing.T) {
	var updatedStatus string
	uc := newJudgeSubmissionUseCase(
		&mockSubmissionUpdater{
			updateFn: func(_ context.Context, s *submission.Submission) error {
				updatedStatus = s.Status().String()
				return nil
			},
		},
		&mockSourceCodeDownloader{},
		&mockProblemProvider{},
		&mockTestCaseProvider{},
		&mockExecutor{},
		&mockOutputChecker{},
	)

	if err := uc.Execute(context.Background(), JudgeSubmissionInput{SubmissionID: submissionID}); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if updatedStatus != "ACCEPTED" {
		t.Errorf("expected ACCEPTED, got %s", updatedStatus)
	}
}

func TestJudgeSubmission_WrongAnswer(t *testing.T) {
	var updatedStatus string
	uc := newJudgeSubmissionUseCase(
		&mockSubmissionUpdater{
			updateFn: func(_ context.Context, s *submission.Submission) error {
				updatedStatus = s.Status().String()
				return nil
			},
		},
		&mockSourceCodeDownloader{},
		&mockProblemProvider{},
		&mockTestCaseProvider{},
		&mockExecutor{},
		&mockOutputChecker{
			checkFn: func(_ context.Context, _ []byte) (CheckResult, error) {
				return CheckResult{Accepted: false}, nil
			},
		},
	)

	if err := uc.Execute(context.Background(), JudgeSubmissionInput{SubmissionID: submissionID}); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if updatedStatus != "WRONG_ANSWER" {
		t.Errorf("expected WRONG_ANSWER, got %s", updatedStatus)
	}
}

func TestJudgeSubmission_TimeLimitExceeded(t *testing.T) {
	var updatedStatus string
	uc := newJudgeSubmissionUseCase(
		&mockSubmissionUpdater{
			updateFn: func(_ context.Context, s *submission.Submission) error {
				updatedStatus = s.Status().String()
				return nil
			},
		},
		&mockSourceCodeDownloader{},
		&mockProblemProvider{},
		&mockTestCaseProvider{},
		&mockExecutor{
			beginSessionFn: func(_ context.Context, _ submission.Language, _ int, _ string) (ExecutionSession, error) {
				return &mockExecutionSession{
					runTestCaseFn: func(_ context.Context, _ RunRequest) (RunResult, error) {
						return RunResult{ExitCode: 124, TimeMs: 1000}, nil
					},
				}, nil
			},
		},
		&mockOutputChecker{},
	)

	if err := uc.Execute(context.Background(), JudgeSubmissionInput{SubmissionID: submissionID}); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if updatedStatus != "TIME_LIMIT_EXCEEDED" {
		t.Errorf("expected TIME_LIMIT_EXCEEDED, got %s", updatedStatus)
	}
}

func TestJudgeSubmission_TimeLimitExceededByCPUTime(t *testing.T) {
	var updatedStatus string
	uc := newJudgeSubmissionUseCase(
		&mockSubmissionUpdater{
			updateFn: func(_ context.Context, s *submission.Submission) error {
				updatedStatus = s.Status().String()
				return nil
			},
		},
		&mockSourceCodeDownloader{},
		&mockProblemProvider{},
		&mockTestCaseProvider{},
		&mockExecutor{
			beginSessionFn: func(_ context.Context, _ submission.Language, _ int, _ string) (ExecutionSession, error) {
				return &mockExecutionSession{
					runTestCaseFn: func(_ context.Context, _ RunRequest) (RunResult, error) {
						return RunResult{ExitCode: 0, TimeMs: 1500, MemoryKb: kb(1024), OutputPreview: []byte("3")}, nil
					},
				}, nil
			},
		},
		&mockOutputChecker{},
	)

	if err := uc.Execute(context.Background(), JudgeSubmissionInput{SubmissionID: submissionID}); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if updatedStatus != "TIME_LIMIT_EXCEEDED" {
		t.Errorf("expected TIME_LIMIT_EXCEEDED, got %s", updatedStatus)
	}
}

func TestJudgeSubmission_MemoryLimitExceeded(t *testing.T) {
	var updatedStatus string
	uc := newJudgeSubmissionUseCase(
		&mockSubmissionUpdater{
			updateFn: func(_ context.Context, s *submission.Submission) error {
				updatedStatus = s.Status().String()
				return nil
			},
		},
		&mockSourceCodeDownloader{},
		&mockProblemProvider{},
		&mockTestCaseProvider{},
		&mockExecutor{
			beginSessionFn: func(_ context.Context, _ submission.Language, _ int, _ string) (ExecutionSession, error) {
				return &mockExecutionSession{
					runTestCaseFn: func(_ context.Context, _ RunRequest) (RunResult, error) {
						return RunResult{ExitCode: 137, MemoryKb: kb(262144)}, nil
					},
				}, nil
			},
		},
		&mockOutputChecker{},
	)

	if err := uc.Execute(context.Background(), JudgeSubmissionInput{SubmissionID: submissionID}); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if updatedStatus != "MEMORY_LIMIT_EXCEEDED" {
		t.Errorf("expected MEMORY_LIMIT_EXCEEDED, got %s", updatedStatus)
	}
}

func TestJudgeSubmission_RuntimeError(t *testing.T) {
	var updatedStatus string
	uc := newJudgeSubmissionUseCase(
		&mockSubmissionUpdater{
			updateFn: func(_ context.Context, s *submission.Submission) error {
				updatedStatus = s.Status().String()
				return nil
			},
		},
		&mockSourceCodeDownloader{},
		&mockProblemProvider{},
		&mockTestCaseProvider{},
		&mockExecutor{
			beginSessionFn: func(_ context.Context, _ submission.Language, _ int, _ string) (ExecutionSession, error) {
				return &mockExecutionSession{
					runTestCaseFn: func(_ context.Context, _ RunRequest) (RunResult, error) {
						return RunResult{ExitCode: 1, TimeMs: 20, MemoryKb: kb(512)}, nil
					},
				}, nil
			},
		},
		&mockOutputChecker{},
	)

	if err := uc.Execute(context.Background(), JudgeSubmissionInput{SubmissionID: submissionID}); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if updatedStatus != "RUNTIME_ERROR" {
		t.Errorf("expected RUNTIME_ERROR, got %s", updatedStatus)
	}
}

func TestJudgeSubmission_SessionInfraError_MarksSystemError(t *testing.T) {
	var updatedStatus string
	uc := newJudgeSubmissionUseCase(
		&mockSubmissionUpdater{
			updateFn: func(_ context.Context, s *submission.Submission) error {
				updatedStatus = s.Status().String()
				return nil
			},
		},
		&mockSourceCodeDownloader{},
		&mockProblemProvider{},
		&mockTestCaseProvider{},
		&mockExecutor{
			beginSessionFn: func(_ context.Context, _ submission.Language, _ int, _ string) (ExecutionSession, error) {
				return &mockExecutionSession{
					runTestCaseFn: func(_ context.Context, _ RunRequest) (RunResult, error) {
						return RunResult{}, errTransient
					},
				}, nil
			},
		},
		&mockOutputChecker{},
	)

	if err := uc.Execute(context.Background(), JudgeSubmissionInput{SubmissionID: submissionID}); err != nil {
		t.Errorf("expected nil (ACK) for infra error after session started, got %v", err)
	}
	if updatedStatus != "SYSTEM_ERROR" {
		t.Errorf("expected SYSTEM_ERROR, got %s", updatedStatus)
	}
}

func TestJudgeSubmission_TxError_ReturnsError(t *testing.T) {
	txCalled := false
	uc := &JudgeSubmissionUseCase{
		submissionUpdater:    &mockSubmissionUpdater{},
		sourceCodeDownloader: &mockSourceCodeDownloader{},
		problemProvider:      &mockProblemProvider{},
		testCaseProvider:     &mockTestCaseProvider{},
		executor:             &mockExecutor{},
		outputChecker:        &mockOutputChecker{},
		txManager:            &mockFailingTxManager{fn: func() { txCalled = true }},
		retry:                RetryConfig{MaxAttempts: 1},
		sleep:                func(time.Duration) {},
	}

	if err := uc.Execute(context.Background(), JudgeSubmissionInput{SubmissionID: submissionID}); err == nil {
		t.Error("expected error when tx fails, got nil")
	}
	if !txCalled {
		t.Error("expected tx to be attempted")
	}
}

// mockFailingTxManager always returns an error from WithTx.
type mockFailingTxManager struct {
	fn func()
}

func (m *mockFailingTxManager) WithTx(_ context.Context, _ func(txCtx context.Context) error) error {
	if m.fn != nil {
		m.fn()
	}
	return errTransient
}

// ── Retry tests ──────────────────────────────────────────────────────────────

func TestJudgeSubmission_TransientError_Retries_ThenSucceeds(t *testing.T) {
	var updatedStatus string
	executor := &mockExecutor{}
	attempt := 0
	executor.beginSessionFn = func(_ context.Context, _ submission.Language, _ int, _ string) (ExecutionSession, error) {
		attempt++
		if attempt < 3 {
			return nil, errors.New("docker unavailable")
		}
		return &mockExecutionSession{}, nil
	}

	uc := newJudgeSubmissionUseCase(
		&mockSubmissionUpdater{
			updateFn: func(_ context.Context, s *submission.Submission) error {
				updatedStatus = s.Status().String()
				return nil
			},
		},
		&mockSourceCodeDownloader{},
		&mockProblemProvider{},
		&mockTestCaseProvider{},
		executor,
		&mockOutputChecker{},
	)
	uc.retry = RetryConfig{MaxAttempts: 3, BackoffBase: 0}

	if err := uc.Execute(context.Background(), JudgeSubmissionInput{SubmissionID: submissionID}); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if updatedStatus != "ACCEPTED" {
		t.Errorf("expected ACCEPTED, got %s", updatedStatus)
	}
	if executor.calls != 3 {
		t.Errorf("expected BeginSession called 3 times, got %d", executor.calls)
	}
}

func TestJudgeSubmission_TransientError_ExhaustsRetries_MarksSystemError(t *testing.T) {
	var updatedStatus string
	executor := &mockExecutor{
		beginSessionFn: func(_ context.Context, _ submission.Language, _ int, _ string) (ExecutionSession, error) {
			return nil, errors.New("docker unavailable")
		},
	}

	uc := newJudgeSubmissionUseCase(
		&mockSubmissionUpdater{
			updateFn: func(_ context.Context, s *submission.Submission) error {
				updatedStatus = s.Status().String()
				return nil
			},
		},
		&mockSourceCodeDownloader{},
		&mockProblemProvider{},
		&mockTestCaseProvider{},
		executor,
		&mockOutputChecker{},
	)
	uc.retry = RetryConfig{MaxAttempts: 3, BackoffBase: 0}

	if err := uc.Execute(context.Background(), JudgeSubmissionInput{SubmissionID: submissionID}); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if updatedStatus != "SYSTEM_ERROR" {
		t.Errorf("expected SYSTEM_ERROR, got %s", updatedStatus)
	}
	if executor.calls != 3 {
		t.Errorf("expected BeginSession called 3 times, got %d", executor.calls)
	}
}

func TestJudgeSubmission_RunTestCaseInfraError_Retries(t *testing.T) {
	var updatedStatus string
	executor := &mockExecutor{}
	sessionCall := 0
	executor.beginSessionFn = func(_ context.Context, _ submission.Language, _ int, _ string) (ExecutionSession, error) {
		sessionCall++
		runCall := 0
		return &mockExecutionSession{
			runTestCaseFn: func(_ context.Context, _ RunRequest) (RunResult, error) {
				runCall++
				if sessionCall == 1 && runCall == 1 {
					return RunResult{}, errors.New("container died")
				}
				return RunResult{ExitCode: 0, TimeMs: 50, MemoryKb: kb(1024), OutputPreview: []byte("3")}, nil
			},
		}, nil
	}

	uc := newJudgeSubmissionUseCase(
		&mockSubmissionUpdater{
			updateFn: func(_ context.Context, s *submission.Submission) error {
				updatedStatus = s.Status().String()
				return nil
			},
		},
		&mockSourceCodeDownloader{},
		&mockProblemProvider{},
		&mockTestCaseProvider{},
		executor,
		&mockOutputChecker{},
	)
	uc.retry = RetryConfig{MaxAttempts: 3, BackoffBase: 0}

	if err := uc.Execute(context.Background(), JudgeSubmissionInput{SubmissionID: submissionID}); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if updatedStatus != "ACCEPTED" {
		t.Errorf("expected ACCEPTED, got %s", updatedStatus)
	}
	if executor.calls != 2 {
		t.Errorf("expected BeginSession called 2 times, got %d", executor.calls)
	}
}

func TestJudgeSubmission_CompilationError_DoesNotRetry(t *testing.T) {
	var updatedStatus string
	executor := &mockExecutor{
		beginSessionFn: func(_ context.Context, _ submission.Language, _ int, _ string) (ExecutionSession, error) {
			return &mockExecutionSession{
				compileFn: func(_ context.Context, _ CompileRequest) (CompileResult, error) {
					return CompileResult{Success: false, Log: "error: undeclared"}, nil
				},
			}, nil
		},
	}

	uc := newJudgeSubmissionUseCase(
		&mockSubmissionUpdater{
			updateFn: func(_ context.Context, s *submission.Submission) error {
				updatedStatus = s.Status().String()
				return nil
			},
		},
		&mockSourceCodeDownloader{},
		&mockProblemProvider{},
		&mockTestCaseProvider{},
		executor,
		&mockOutputChecker{},
	)
	uc.retry = RetryConfig{MaxAttempts: 3, BackoffBase: 0}

	if err := uc.Execute(context.Background(), JudgeSubmissionInput{SubmissionID: submissionID}); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if updatedStatus != "COMPILATION_ERROR" {
		t.Errorf("expected COMPILATION_ERROR, got %s", updatedStatus)
	}
	if executor.calls != 1 {
		t.Errorf("expected BeginSession called 1 time, got %d", executor.calls)
	}
}

func TestJudgeSubmission_WrongAnswer_DoesNotRetry(t *testing.T) {
	var updatedStatus string
	checkerCalls := 0
	executor := &mockExecutor{}

	uc := newJudgeSubmissionUseCase(
		&mockSubmissionUpdater{
			updateFn: func(_ context.Context, s *submission.Submission) error {
				updatedStatus = s.Status().String()
				return nil
			},
		},
		&mockSourceCodeDownloader{},
		&mockProblemProvider{},
		&mockTestCaseProvider{},
		executor,
		&mockOutputChecker{
			checkFn: func(_ context.Context, _ []byte) (CheckResult, error) {
				checkerCalls++
				return CheckResult{Accepted: false}, nil
			},
		},
	)
	uc.retry = RetryConfig{MaxAttempts: 3, BackoffBase: 0}

	if err := uc.Execute(context.Background(), JudgeSubmissionInput{SubmissionID: submissionID}); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if updatedStatus != "WRONG_ANSWER" {
		t.Errorf("expected WRONG_ANSWER, got %s", updatedStatus)
	}
	if checkerCalls != 1 {
		t.Errorf("expected checker called 1 time, got %d", checkerCalls)
	}
}

// Same contract as the publish path: one session per attempt, always closed.
func TestJudgeSubmission_OpensAndClosesOneCheckerSession(t *testing.T) {
	begins := 0
	checker := &mockOutputChecker{}
	checker.beginCheckingFn = func(context.Context, string, submission.Language, string) (CheckerSession, error) {
		begins++
		checker.session = &mockCheckerSession{}
		return checker.session, nil
	}
	testCases := &mockTestCaseProvider{
		getTestCasesFn: func(context.Context, string) ([]TestCase, error) {
			return []TestCase{
				{Input: []byte("1 2"), ExpectedOutput: []byte("3")},
				{Input: []byte("2 3"), ExpectedOutput: []byte("5")},
			}, nil
		},
	}
	sub := pendingSubmission()
	uc := newJudgeSubmissionUseCase(
		&mockSubmissionUpdater{
			getByIDFn: func(context.Context, submission.SubmissionID) (*submission.Submission, error) {
				return sub, nil
			},
		},
		&mockSourceCodeDownloader{}, &mockProblemProvider{}, testCases, &mockExecutor{}, checker,
	)

	if err := uc.Execute(context.Background(), JudgeSubmissionInput{SubmissionID: submissionID}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if begins != 1 {
		t.Errorf("BeginChecking calls: got %d, want 1 for two test cases", begins)
	}
	if checker.session.checkCalls != 2 {
		t.Errorf("Check calls: got %d, want one per test case", checker.session.checkCalls)
	}
	if checker.session.closeCalls != 1 {
		t.Errorf("Close calls: got %d, want 1 — an unclosed session leaks its container", checker.session.closeCalls)
	}
}

// The problem's memory limit moved from RunRequest (per test case, where nobody
// read it) to BeginSession (per judging, where the container is configured).
// That is the shape of the CheckerLanguage bug: a field set at one call site and
// forgotten at the other. Every other test here mocks the executor without
// looking at its arguments, so only this one would notice it stop arriving.
func TestJudgeSubmission_ProblemMemoryLimitReachesTheSession(t *testing.T) {
	const wantMemoryKb = 131072 // 128 MB, different from the mock's default

	gotMemoryKb := -1
	uc := newJudgeSubmissionUseCase(
		&mockSubmissionUpdater{},
		&mockSourceCodeDownloader{},
		&mockProblemProvider{
			getLimitsFn: func(_ context.Context, _ string) (ProblemLimits, error) {
				return ProblemLimits{TimeLimitMs: 1000, MemoryKb: wantMemoryKb}, nil
			},
		},
		&mockTestCaseProvider{},
		&mockExecutor{
			beginSessionFn: func(_ context.Context, _ submission.Language, memoryKb int, _ string) (ExecutionSession, error) {
				gotMemoryKb = memoryKb
				return &mockExecutionSession{}, nil
			},
		},
		&mockOutputChecker{},
	)

	if err := uc.Execute(context.Background(), JudgeSubmissionInput{SubmissionID: submissionID}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotMemoryKb != wantMemoryKb {
		t.Errorf("BeginSession got memoryKb = %d, want the problem's %d", gotMemoryKb, wantMemoryKb)
	}
}

// Both sandboxes have to be pointed at the same directory: the heavy pool writes
// the contestant's output there and the checker reads it. A mismatch compiles
// perfectly and makes every checker read an empty file.
func TestJudgeSubmission_BothSessionsShareOneJudgingDirectory(t *testing.T) {
	var executorID, checkerID string
	executor := &mockExecutor{
		beginSessionFn: func(_ context.Context, _ submission.Language, _ int, judgingID string) (ExecutionSession, error) {
			executorID = judgingID
			return &mockExecutionSession{}, nil
		},
	}
	checker := &mockOutputChecker{
		beginCheckingFn: func(_ context.Context, _ string, _ submission.Language, judgingID string) (CheckerSession, error) {
			checkerID = judgingID
			return &mockCheckerSession{}, nil
		},
	}
	uc := newJudgeSubmissionUseCase(
		&mockSubmissionUpdater{}, &mockSourceCodeDownloader{}, &mockProblemProvider{},
		&mockTestCaseProvider{}, executor, checker,
	)

	if err := uc.Execute(context.Background(), JudgeSubmissionInput{SubmissionID: submissionID}); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	if executorID == "" {
		t.Fatal("the execution session was opened without a judging directory")
	}
	if checkerID != executorID {
		t.Errorf("checker got %q, the executor got %q — they must share one directory", checkerID, executorID)
	}
}

// The directory name is the credential: the volume root is not listable, so a
// sandbox reaches only the path it was handed. A name that repeats, or one a
// contestant can look up — the submission id is public over the API — hands
// every judging to anyone who asks.
func TestJudgeSubmission_TheJudgingDirectoryIsUnguessable(t *testing.T) {
	var ids []string
	runOnce := func() {
		executor := &mockExecutor{
			beginSessionFn: func(_ context.Context, _ submission.Language, _ int, judgingID string) (ExecutionSession, error) {
				ids = append(ids, judgingID)
				return &mockExecutionSession{}, nil
			},
		}
		uc := newJudgeSubmissionUseCase(
			&mockSubmissionUpdater{}, &mockSourceCodeDownloader{}, &mockProblemProvider{},
			&mockTestCaseProvider{}, executor, &mockOutputChecker{},
		)
		if err := uc.Execute(context.Background(), JudgeSubmissionInput{SubmissionID: submissionID}); err != nil {
			t.Fatalf("Execute: %v", err)
		}
	}
	runOnce()
	runOnce()

	if len(ids) != 2 {
		t.Fatalf("expected two judgings, got %d", len(ids))
	}
	if ids[0] == ids[1] {
		t.Errorf("both judgings used %q: a fixed name lets one judging read another", ids[0])
	}
	for _, id := range ids {
		if id == submissionID {
			t.Errorf("the judging directory is the submission id, which the API hands out")
		}
	}
}

// The verdict reports the largest measurement across the test cases, and
// reports nothing at all when there is none — a zero would say the solution
// used no memory, which is exactly what the old cgroup field did every time.
func TestJudgeSubmission_ReportsTheLargestMeasurementOrNone(t *testing.T) {
	tests := []struct {
		name    string
		perCase []*int
		wantKb  *int
	}{
		{"the largest of what was measured", []*int{kb(512), kb(4096), kb(1024)}, kb(4096)},
		{"some cases measured, some not", []*int{nil, kb(2048), nil}, kb(2048)},
		{"nothing measured at all", []*int{nil, nil, nil}, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sub := pendingSubmission()
			i := 0
			executor := &mockExecutor{
				beginSessionFn: func(_ context.Context, _ submission.Language, _ int, _ string) (ExecutionSession, error) {
					return &mockExecutionSession{
						runTestCaseFn: func(context.Context, RunRequest) (RunResult, error) {
							r := RunResult{ExitCode: 0, TimeMs: 10, MemoryKb: tt.perCase[i], OutputPreview: []byte("3")}
							i++
							return r, nil
						},
					}, nil
				},
			}
			testCases := &mockTestCaseProvider{
				getTestCasesFn: func(context.Context, string) ([]TestCase, error) {
					return []TestCase{
						{Input: []byte("1"), ExpectedOutput: []byte("3")},
						{Input: []byte("2"), ExpectedOutput: []byte("3")},
						{Input: []byte("3"), ExpectedOutput: []byte("3")},
					}, nil
				},
			}
			uc := newJudgeSubmissionUseCase(
				&mockSubmissionUpdater{
					getByIDFn: func(context.Context, submission.SubmissionID) (*submission.Submission, error) {
						return sub, nil
					},
				},
				&mockSourceCodeDownloader{}, &mockProblemProvider{},
				testCases, executor, &mockOutputChecker{},
			)

			if err := uc.Execute(context.Background(), JudgeSubmissionInput{SubmissionID: submissionID}); err != nil {
				t.Fatalf("Execute: %v", err)
			}

			got := sub.MemoryKb()
			switch {
			case tt.wantKb == nil && got != nil:
				t.Errorf("MemoryKb = %d, want nil", *got)
			case tt.wantKb != nil && got == nil:
				t.Errorf("MemoryKb is nil, want %d", *tt.wantKb)
			case tt.wantKb != nil && *got != *tt.wantKb:
				t.Errorf("MemoryKb = %d, want %d", *got, *tt.wantKb)
			}
		})
	}
}
