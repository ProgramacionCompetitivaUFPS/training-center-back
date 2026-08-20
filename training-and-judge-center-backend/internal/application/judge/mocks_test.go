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
	calls          int
	beginSessionFn func(ctx context.Context, language submission.Language) (ExecutionSession, error)
}

func (m *mockExecutor) BeginSession(ctx context.Context, language submission.Language) (ExecutionSession, error) {
	m.calls++
	if m.beginSessionFn != nil {
		return m.beginSessionFn(ctx, language)
	}
	return &mockExecutionSession{}, nil
}

// ── ExecutionSession mock ────────────────────────────────────────────────────

type mockExecutionSession struct {
	compileFn     func(ctx context.Context, req CompileRequest) (CompileResult, error)
	runTestCaseFn func(ctx context.Context, req RunRequest) (RunResult, error)
	closeFn       func(ctx context.Context) error
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
	beginCheckingFn func(ctx context.Context, checkerPath string, language submission.Language) (CheckerSession, error)
	// checkFn is a shortcut for the common case: a session that always answers
	// the same way, without having to build one.
	checkFn func(ctx context.Context, req CheckRequest) (CheckResult, error)
	session *mockCheckerSession
}

func (m *mockOutputChecker) BeginChecking(ctx context.Context, checkerPath string, language submission.Language) (CheckerSession, error) {
	if m.beginCheckingFn != nil {
		return m.beginCheckingFn(ctx, checkerPath, language)
	}
	if m.session == nil {
		m.session = &mockCheckerSession{checkFn: m.checkFn}
	}
	return m.session, nil
}

// ── CheckerSession mock ──────────────────────────────────────────────────────

type mockCheckerSession struct {
	checkFn    func(ctx context.Context, req CheckRequest) (CheckResult, error)
	checkCalls int
	closeCalls int
}

func (m *mockCheckerSession) Check(ctx context.Context, req CheckRequest) (CheckResult, error) {
	m.checkCalls++
	if m.checkFn != nil {
		return m.checkFn(ctx, req)
	}
	return CheckResult{Accepted: true}, nil
}

func (m *mockCheckerSession) Close(context.Context) error {
	m.closeCalls++
	return nil
}

// ── JudgingSourceProvider mock ───────────────────────────────────────────────

type mockJudgingSourceProvider struct {
	getCheckerSourceFn   func(ctx context.Context, problemID string) (*JudgingSource, error)
	getValidatorSourceFn func(ctx context.Context, problemID string) (*JudgingSource, error)
}

func (m *mockJudgingSourceProvider) GetCheckerSource(ctx context.Context, problemID string) (*JudgingSource, error) {
	if m.getCheckerSourceFn != nil {
		return m.getCheckerSourceFn(ctx, problemID)
	}
	return nil, nil
}

func (m *mockJudgingSourceProvider) GetValidatorSource(ctx context.Context, problemID string) (*JudgingSource, error) {
	if m.getValidatorSourceFn != nil {
		return m.getValidatorSourceFn(ctx, problemID)
	}
	return nil, nil
}

// ── ArtifactCompiler mock ──────────────────────────────────────────────────────

type mockArtifactCompiler struct {
	compileFn func(ctx context.Context, req CompileArtifactRequest) (CompileArtifactResult, error)
}

func (m *mockArtifactCompiler) Compile(ctx context.Context, req CompileArtifactRequest) (CompileArtifactResult, error) {
	if m.compileFn != nil {
		return m.compileFn(ctx, req)
	}
	return CompileArtifactResult{Success: true, Artifact: []byte("artifact")}, nil
}

// ── ArtifactUploader mock ────────────────────────────────────────────────────

type mockArtifactUploader struct {
	uploadFn func(ctx context.Context, path string, content []byte) error
	uploaded map[string][]byte
}

func (m *mockArtifactUploader) Upload(ctx context.Context, path string, content []byte) error {
	if m.uploaded == nil {
		m.uploaded = map[string][]byte{}
	}
	m.uploaded[path] = content
	if m.uploadFn != nil {
		return m.uploadFn(ctx, path, content)
	}
	return nil
}

// ── ValidatorRunner mock ─────────────────────────────────────────────────────

type mockValidatorRunner struct {
	beginValidatingFn func(ctx context.Context, validatorPath string, language submission.Language) (ValidatorSession, error)
	session           *mockValidatorSession
}

func (m *mockValidatorRunner) BeginValidating(ctx context.Context, validatorPath string, language submission.Language) (ValidatorSession, error) {
	if m.beginValidatingFn != nil {
		return m.beginValidatingFn(ctx, validatorPath, language)
	}
	if m.session == nil {
		m.session = &mockValidatorSession{}
	}
	return m.session, nil
}

// ── ValidatorSession mock ────────────────────────────────────────────────────

type mockValidatorSession struct {
	validateFn func(ctx context.Context, input []byte) (ValidatorRunResult, error)
	closeCalls int
}

func (m *mockValidatorSession) Validate(ctx context.Context, input []byte) (ValidatorRunResult, error) {
	if m.validateFn != nil {
		return m.validateFn(ctx, input)
	}
	return ValidatorRunResult{Accepted: true}, nil
}

func (m *mockValidatorSession) Close(ctx context.Context) error {
	m.closeCalls++
	return nil
}

// ── SolutionProvider mock ────────────────────────────────────────────────────

type mockSolutionProvider struct {
	getSolutionsFn func(ctx context.Context, problemID string) ([]Solution, error)
}

func (m *mockSolutionProvider) GetSolutions(ctx context.Context, problemID string) ([]Solution, error) {
	if m.getSolutionsFn != nil {
		return m.getSolutionsFn(ctx, problemID)
	}
	return []Solution{{FileKey: "solutions/sol.cpp", Language: submission.RestoreLanguage("cpp20")}}, nil
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

// ── StaleSubmissionRecoverer mock ────────────────────────────────────────────

type mockStaleSubmissionRecoverer struct {
	recoverFn func(ctx context.Context, cutoff time.Time) (int, error)
}

func (m *mockStaleSubmissionRecoverer) RecoverStaleBefore(ctx context.Context, cutoff time.Time) (int, error) {
	if m.recoverFn != nil {
		return m.recoverFn(ctx, cutoff)
	}
	return 0, nil
}

// ── Test constants ───────────────────────────────────────────────────────────

const (
	submissionID = "aaaaaaaa-0000-0000-0000-000000000001"
	problemID    = "bbbbbbbb-0000-0000-0000-000000000001"
	userID       = "cccccccc-0000-0000-0000-000000000001"
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
		submission.RestoreVisibility("PRIVATE"),
		"gs://bucket/code.cpp",
		"", 0,
		testNow, nil, nil, nil, nil, "", "",
	)
}

var errTransient = errors.New("transient error")
