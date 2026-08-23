package judge

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	appjudge "github.com/training-judge-center/backend/internal/application/judge"
	appshared "github.com/training-judge-center/backend/internal/application/shared"
	"github.com/training-judge-center/backend/internal/domain/shared"
	"github.com/training-judge-center/backend/internal/domain/submission"
)

// The judging use case and this adapter used to be tested on opposite sides of
// the same seam, and the two fields naming the checker got lost in between: the
// real judging path built its request without them, so every submission to a
// problem with a custom checker ended in SYSTEM_ERROR. The unit tests could not
// see it — one mocks the checker, the other calls the adapter with the fields
// already filled in. This test crosses the seam, with the real use case over the
// real adapter and only their neighbours mocked.

const (
	boundarySubmissionID = "aaaaaaaa-0000-0000-0000-000000000001"
	boundaryProblemID    = "bbbbbbbb-0000-0000-0000-000000000001"
	boundaryUserID       = "cccccccc-0000-0000-0000-000000000001"
)

func pendingBoundarySubmission() *submission.Submission {
	return submission.RestoreSubmission(
		boundarySubmissionID, boundaryProblemID,
		shared.RestoreUserID(boundaryUserID),
		nil, nil,
		submission.RestoreLanguage(testLang), "g++",
		submission.RestoreStatus("PENDING"),
		submission.RestoreVisibility("PRIVATE"),
		"gs://bucket/code.cpp",
		"", 0,
		time.Now(), nil, nil, nil, nil, "", "",
	)
}

type boundarySubmissionUpdater struct {
	sub *submission.Submission
}

func (m *boundarySubmissionUpdater) GetByID(context.Context, submission.SubmissionID) (*submission.Submission, error) {
	return m.sub, nil
}
func (m *boundarySubmissionUpdater) Update(context.Context, *submission.Submission) error { return nil }

type boundaryDownloader struct{}

func (boundaryDownloader) Download(context.Context, string) ([]byte, error) {
	return []byte("int main(){}"), nil
}

type boundaryProblemProvider struct {
	limits appjudge.ProblemLimits
}

func (m *boundaryProblemProvider) GetLimits(context.Context, string) (appjudge.ProblemLimits, error) {
	return m.limits, nil
}

type boundaryTestCaseProvider struct{}

func (boundaryTestCaseProvider) GetTestCases(context.Context, string) ([]appjudge.TestCase, error) {
	return []appjudge.TestCase{
		{Name: "secret/001", Input: []byte("1 2"), ExpectedOutput: []byte("3")},
	}, nil
}

// boundaryExecutor stands for the heavy pool. It leaves the contestant's output
// in the judging directory it was named, which is the only place the checker
// under test can find it: the two sides agreeing on that path is the seam.
type boundaryExecutor struct {
	root string
	dir  string // recorded so the test can assert what the checker was pointed at
}

func (e *boundaryExecutor) BeginSession(_ context.Context, _ submission.Language, _ int, judgingID string) (appjudge.ExecutionSession, error) {
	e.dir = filepath.Join(e.root, judgingID)
	if err := os.Mkdir(e.dir, 0o755); err != nil {
		return nil, err
	}
	return boundaryExecutionSession{dir: e.dir}, nil
}

type boundaryExecutionSession struct{ dir string }

func (boundaryExecutionSession) Compile(context.Context, appjudge.CompileRequest) (appjudge.CompileResult, error) {
	return appjudge.CompileResult{Success: true}, nil
}

func (s boundaryExecutionSession) RunTestCase(context.Context, appjudge.RunRequest) (appjudge.RunResult, error) {
	if err := os.WriteFile(judgingOutputPath(s.dir), []byte("3"), judgingFileMode); err != nil {
		return appjudge.RunResult{}, err
	}
	return appjudge.RunResult{ExitCode: 0, TimeMs: 10, MemoryKb: 1024, OutputPreview: []byte("3")}, nil
}

func (boundaryExecutionSession) Close(context.Context) error { return nil }

type boundaryTxManager struct{}

func (boundaryTxManager) WithTx(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}

var _ appshared.TransactionManager = boundaryTxManager{}

func TestJudgeSubmission_CustomChecker_RunsTheCheckerInTheSandbox(t *testing.T) {
	var cmds [][]string
	docker := &mockDockerExecClient{}
	recordExecs(docker, &cmds)
	root := t.TempDir()
	checker, poolDocker := newTestOutputChecker(t, docker, storedArtifact("ELF binary"), root)
	executor := &boundaryExecutor{root: root}

	sub := pendingBoundarySubmission()
	uc := appjudge.NewJudgeSubmissionUseCase(
		&boundarySubmissionUpdater{sub: sub},
		boundaryDownloader{},
		&boundaryProblemProvider{limits: appjudge.ProblemLimits{
			TimeLimitMs:      1000,
			MemoryKb:         262144,
			HasCustomChecker: true,
			CheckerPath:      testCheckerKey,
			CheckerLanguage:  submission.RestoreLanguage(testLang),
		}},
		boundaryTestCaseProvider{},
		executor,
		checker,
		boundaryTxManager{},
		appjudge.RetryConfig{MaxAttempts: 1},
	)

	if err := uc.Execute(context.Background(), appjudge.JudgeSubmissionInput{SubmissionID: boundarySubmissionID}); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	if got := poolDocker.idCounter.Load(); got != 1 {
		t.Fatalf("expected the checker to claim one light pool container, the pool created %d", got)
	}

	// The paths name the directory the executor created: the checker reading a
	// different one is exactly the failure this test exists to catch.
	want := "/sandbox/Checker " + judgingInputPath(executor.dir) + " " +
		judgingOutputPath(executor.dir) + " /sandbox/answer.txt"
	var ran bool
	for _, cmd := range cmds {
		if len(cmd) == 3 && strings.Contains(cmd[2], want) {
			ran = true
		}
	}
	if !ran {
		t.Errorf("expected the custom checker to run, commands were: %v", cmds)
	}

	if got := sub.Status().String(); got != "ACCEPTED" {
		t.Errorf("verdict: got %q, want ACCEPTED — the checker exited 0", got)
	}
}

// The same path without a custom checker now reaches the light pool too, with
// our own comparison binary: nothing about the contestant output travels
// through the worker any more, whichever checker the problem declares.
func TestJudgeSubmission_NoCustomChecker_RunsCompareInTheLightPool(t *testing.T) {
	root := t.TempDir()
	checker, poolDocker := newTestOutputChecker(t, &mockDockerExecClient{}, storedArtifact("ELF binary"), root)
	executor := &boundaryExecutor{root: root}

	sub := pendingBoundarySubmission()
	uc := appjudge.NewJudgeSubmissionUseCase(
		&boundarySubmissionUpdater{sub: sub},
		boundaryDownloader{},
		&boundaryProblemProvider{limits: appjudge.ProblemLimits{TimeLimitMs: 1000, MemoryKb: 262144}},
		boundaryTestCaseProvider{},
		executor,
		checker,
		boundaryTxManager{},
		appjudge.RetryConfig{MaxAttempts: 1},
	)

	if err := uc.Execute(context.Background(), appjudge.JudgeSubmissionInput{SubmissionID: boundarySubmissionID}); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	if got := poolDocker.idCounter.Load(); got != 1 {
		t.Errorf("expected one light pool container, the pool created %d", got)
	}
	if got := poolDocker.lastCreateImage.Load(); got != "judge-runner:compare" {
		t.Errorf("image: got %v, want the compare image", got)
	}
	if got := sub.Status().String(); got != "ACCEPTED" {
		t.Errorf("verdict: got %q, want ACCEPTED — the checker exited 0", got)
	}
}
