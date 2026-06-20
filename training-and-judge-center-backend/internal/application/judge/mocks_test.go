package judge

import (
	"context"
	"errors"
	"time"

	"github.com/training-judge-center/backend/internal/domain/shared"
	"github.com/training-judge-center/backend/internal/domain/submission"
)

var testNow = time.Date(2025, 1, 15, 12, 0, 0, 0, time.UTC)

// ── SubmissionUpdater mock ───────────────────────────────────────────────────

type mockSubmissionUpdater struct {
	getByIDFn func(ctx context.Context, id submission.SubmissionID) (*submission.Submission, error)
	updateFn  func(ctx context.Context, s *submission.Submission) error
}

func (m *mockSubmissionUpdater) GetByID(ctx context.Context, id submission.SubmissionID) (*submission.Submission, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return pendingSubmission(), nil
}

func (m *mockSubmissionUpdater) Update(ctx context.Context, s *submission.Submission) error {
	if m.updateFn != nil {
		return m.updateFn(ctx, s)
	}
	return nil
}

// ── SourceCodeDownloader mock ────────────────────────────────────────────────

type mockSourceCodeDownloader struct {
	downloadFn func(ctx context.Context, path string) ([]byte, error)
}

func (m *mockSourceCodeDownloader) Download(ctx context.Context, path string) ([]byte, error) {
	if m.downloadFn != nil {
		return m.downloadFn(ctx, path)
	}
	return []byte("int main(){}"), nil
}

// ── ProblemProvider mock ─────────────────────────────────────────────────────

type mockProblemProvider struct {
	getLimitsFn func(ctx context.Context, problemID string) (ProblemLimits, error)
}

func (m *mockProblemProvider) GetLimits(ctx context.Context, problemID string) (ProblemLimits, error) {
	if m.getLimitsFn != nil {
		return m.getLimitsFn(ctx, problemID)
	}
	return ProblemLimits{TimeLimitMs: 1000, MemoryKb: 262144}, nil
}

// ── TestCaseProvider mock ────────────────────────────────────────────────────

type mockTestCaseProvider struct {
	getTestCasesFn func(ctx context.Context, problemID string) ([]TestCase, error)
}

func (m *mockTestCaseProvider) GetTestCases(ctx context.Context, problemID string) ([]TestCase, error) {
	if m.getTestCasesFn != nil {
		return m.getTestCasesFn(ctx, problemID)
	}
	return []TestCase{{Input: []byte("1 2"), ExpectedOutput: []byte("3")}}, nil
}

// ── Executor mock ────────────────────────────────────────────────────────────

type mockExecutor struct {
	beginSessionFn func(ctx context.Context, language submission.Language) (ExecutionSession, error)
}

func (m *mockExecutor) BeginSession(ctx context.Context, language submission.Language) (ExecutionSession, error) {
	if m.beginSessionFn != nil {
		return m.beginSessionFn(ctx, language)
	}
	return &mockExecutionSession{}, nil
}

// ── ExecutionSession mock ────────────────────────────────────────────────────

type mockExecutionSession struct {
	compileFn      func(ctx context.Context, req CompileRequest) (CompileResult, error)
	runTestCaseFn  func(ctx context.Context, req RunRequest) (RunResult, error)
	closeFn        func(ctx context.Context) error
}

func (m *mockExecutionSession) Compile(ctx context.Context, req CompileRequest) (CompileResult, error) {
	if m.compileFn != nil {
		return m.compileFn(ctx, req)
	}
	return CompileResult{Success: true}, nil
}

func (m *mockExecutionSession) RunTestCase(ctx context.Context, req RunRequest) (RunResult, error) {
	if m.runTestCaseFn != nil {
		return m.runTestCaseFn(ctx, req)
	}
	return RunResult{ExitCode: 0, TimeMs: 50, MemoryKb: 1024, Output: []byte("3")}, nil
}

func (m *mockExecutionSession) Close(ctx context.Context) error {
	if m.closeFn != nil {
		return m.closeFn(ctx)
	}
	return nil
}

// ── OutputChecker mock ───────────────────────────────────────────────────────

type mockOutputChecker struct {
	checkFn func(ctx context.Context, req CheckRequest) (CheckResult, error)
}

func (m *mockOutputChecker) Check(ctx context.Context, req CheckRequest) (CheckResult, error) {
	if m.checkFn != nil {
		return m.checkFn(ctx, req)
	}
	return CheckResult{Accepted: true}, nil
}

// ── StandingUpdater mock ─────────────────────────────────────────────────────

type mockStandingUpdater struct {
	recordVerdictFn func(ctx context.Context, req RecordVerdictRequest) error
}

func (m *mockStandingUpdater) RecordVerdict(ctx context.Context, req RecordVerdictRequest) error {
	if m.recordVerdictFn != nil {
		return m.recordVerdictFn(ctx, req)
	}
	return nil
}

// ── TransactionManager mock ──────────────────────────────────────────────────

type mockTransactionManager struct {
	withTxFn func(ctx context.Context, fn func(txCtx context.Context) error) error
}

func (m *mockTransactionManager) WithTx(ctx context.Context, fn func(txCtx context.Context) error) error {
	if m.withTxFn != nil {
		return m.withTxFn(ctx, fn)
	}
	return fn(ctx)
}

// ── Test constants ───────────────────────────────────────────────────────────

const (
	submissionID = "aaaaaaaa-0000-0000-0000-000000000001"
	problemID    = "bbbbbbbb-0000-0000-0000-000000000001"
	userID       = "cccccccc-0000-0000-0000-000000000001"
	contestID    = "dddddddd-0000-0000-0000-000000000001"
)

// ── Domain fixtures ──────────────────────────────────────────────────────────

func pendingSubmission() *submission.Submission {
	lang := submission.RestoreLanguage("cpp20")
	return submission.RestoreSubmission(
		submissionID, problemID,
		shared.RestoreUserID(userID),
		nil, nil,
		lang, "g++",
		submission.RestoreStatus("PENDING"),
		"gs://bucket/code.cpp",
		"", 0,
		testNow, nil, nil, nil, nil,
	)
}

func pendingSubmissionInContest(cID string) *submission.Submission {
	lang := submission.RestoreLanguage("cpp20")
	return submission.RestoreSubmission(
		submissionID, problemID,
		shared.RestoreUserID(userID),
		&cID, nil,
		lang, "g++",
		submission.RestoreStatus("PENDING"),
		"gs://bucket/code.cpp",
		"", 0,
		testNow, nil, nil, nil, nil,
	)
}

var errTransient = errors.New("transient error")
