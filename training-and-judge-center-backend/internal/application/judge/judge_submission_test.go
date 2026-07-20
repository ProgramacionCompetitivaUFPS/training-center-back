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
			beginSessionFn: func(_ context.Context, _ submission.Language) (ExecutionSession, error) {
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
			beginSessionFn: func(_ context.Context, _ submission.Language) (ExecutionSession, error) {
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
			checkFn: func(_ context.Context, _ CheckRequest) (CheckResult, error) {
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
			beginSessionFn: func(_ context.Context, _ submission.Language) (ExecutionSession, error) {
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
			beginSessionFn: func(_ context.Context, _ submission.Language) (ExecutionSession, error) {
				return &mockExecutionSession{
					runTestCaseFn: func(_ context.Context, _ RunRequest) (RunResult, error) {
						return RunResult{ExitCode: 0, TimeMs: 1500, MemoryKb: 1024, Output: []byte("3")}, nil
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
			beginSessionFn: func(_ context.Context, _ submission.Language) (ExecutionSession, error) {
				return &mockExecutionSession{
					runTestCaseFn: func(_ context.Context, _ RunRequest) (RunResult, error) {
						return RunResult{ExitCode: 137, MemoryKb: 262144}, nil
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
			beginSessionFn: func(_ context.Context, _ submission.Language) (ExecutionSession, error) {
				return &mockExecutionSession{
					runTestCaseFn: func(_ context.Context, _ RunRequest) (RunResult, error) {
						return RunResult{ExitCode: 1, TimeMs: 20, MemoryKb: 512}, nil
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
			beginSessionFn: func(_ context.Context, _ submission.Language) (ExecutionSession, error) {
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
	executor.beginSessionFn = func(_ context.Context, _ submission.Language) (ExecutionSession, error) {
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
		beginSessionFn: func(_ context.Context, _ submission.Language) (ExecutionSession, error) {
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
	executor.beginSessionFn = func(_ context.Context, _ submission.Language) (ExecutionSession, error) {
		sessionCall++
		runCall := 0
		return &mockExecutionSession{
			runTestCaseFn: func(_ context.Context, _ RunRequest) (RunResult, error) {
				runCall++
				if sessionCall == 1 && runCall == 1 {
					return RunResult{}, errors.New("container died")
				}
				return RunResult{ExitCode: 0, TimeMs: 50, MemoryKb: 1024, Output: []byte("3")}, nil
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
		beginSessionFn: func(_ context.Context, _ submission.Language) (ExecutionSession, error) {
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
			checkFn: func(_ context.Context, _ CheckRequest) (CheckResult, error) {
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
