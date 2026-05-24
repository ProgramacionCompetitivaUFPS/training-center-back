package judge

import (
	"context"
	"testing"

	"github.com/training-judge-center/backend/internal/domain/submission"
)

func newJudgeSubmissionUseCase(
	updater *mockSubmissionUpdater,
	downloader *mockSourceCodeDownloader,
	problems *mockProblemProvider,
	testCases *mockTestCaseProvider,
	executor *mockExecutor,
	checker *mockOutputChecker,
	standing *mockStandingUpdater,
) *JudgeSubmissionUseCase {
	return NewJudgeSubmissionUseCase(
		updater, downloader, problems, testCases,
		executor, checker, standing, &mockTransactionManager{},
	)
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
		&mockStandingUpdater{},
	)

	if err := uc.Execute(context.Background(), JudgeSubmissionInput{SubmissionID: submissionID}); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if executorCalled {
		t.Error("executor should not be called for non-pending submission")
	}
}

func TestJudgeSubmission_SourceCodeDownloadError_ReturnsError(t *testing.T) {
	uc := newJudgeSubmissionUseCase(
		&mockSubmissionUpdater{},
		&mockSourceCodeDownloader{
			downloadFn: func(_ context.Context, _ string) ([]byte, error) {
				return nil, errTransient
			},
		},
		&mockProblemProvider{},
		&mockTestCaseProvider{},
		&mockExecutor{},
		&mockOutputChecker{},
		&mockStandingUpdater{},
	)

	if err := uc.Execute(context.Background(), JudgeSubmissionInput{SubmissionID: submissionID}); err == nil {
		t.Error("expected error for download failure, got nil")
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
		&mockStandingUpdater{},
	)

	if err := uc.Execute(context.Background(), JudgeSubmissionInput{SubmissionID: submissionID}); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if updatedStatus != "COMPILATION_ERROR" {
		t.Errorf("expected COMPILATION_ERROR, got %s", updatedStatus)
	}
}

func TestJudgeSubmission_Accepted_NoContest(t *testing.T) {
	var updatedStatus string
	recordVerdictCalled := false

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
		&mockStandingUpdater{
			recordVerdictFn: func(_ context.Context, _ RecordVerdictRequest) error {
				recordVerdictCalled = true
				return nil
			},
		},
	)

	if err := uc.Execute(context.Background(), JudgeSubmissionInput{SubmissionID: submissionID}); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if updatedStatus != "ACCEPTED" {
		t.Errorf("expected ACCEPTED, got %s", updatedStatus)
	}
	if recordVerdictCalled {
		t.Error("RecordVerdict should not be called when there is no contest")
	}
}

func TestJudgeSubmission_Accepted_WithContest(t *testing.T) {
	var updatedStatus string
	var recordedContestID string

	uc := newJudgeSubmissionUseCase(
		&mockSubmissionUpdater{
			getByIDFn: func(_ context.Context, _ submission.SubmissionID) (*submission.Submission, error) {
				return pendingSubmissionInContest(contestID), nil
			},
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
		&mockStandingUpdater{
			recordVerdictFn: func(_ context.Context, req RecordVerdictRequest) error {
				recordedContestID = req.ContestID
				return nil
			},
		},
	)

	if err := uc.Execute(context.Background(), JudgeSubmissionInput{SubmissionID: submissionID}); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if updatedStatus != "ACCEPTED" {
		t.Errorf("expected ACCEPTED, got %s", updatedStatus)
	}
	if recordedContestID != contestID {
		t.Errorf("expected RecordVerdict with contestID %s, got %s", contestID, recordedContestID)
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
		&mockStandingUpdater{},
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
		&mockStandingUpdater{},
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
		&mockStandingUpdater{},
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
		&mockStandingUpdater{},
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
		&mockStandingUpdater{},
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
		standingUpdater:      &mockStandingUpdater{},
		txManager:            &mockFailingTxManager{fn: func() { txCalled = true }},
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
