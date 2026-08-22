package judge

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/moby/moby/client"
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

// boundaryExecutor stands for the heavy pool: it compiles fine and produces the
// contestant's output, which is what the checker under test then receives.
type boundaryExecutor struct{}

func (boundaryExecutor) BeginSession(context.Context, submission.Language, int) (appjudge.ExecutionSession, error) {
	return boundaryExecutionSession{}, nil
}

type boundaryExecutionSession struct{}

func (boundaryExecutionSession) Compile(context.Context, appjudge.CompileRequest) (appjudge.CompileResult, error) {
	return appjudge.CompileResult{Success: true}, nil
}

func (boundaryExecutionSession) RunTestCase(context.Context, appjudge.RunRequest) (appjudge.RunResult, error) {
	return appjudge.RunResult{ExitCode: 0, TimeMs: 10, MemoryKb: 1024, Output: []byte("3")}, nil
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
	checker, poolDocker := newTestOutputChecker(t, docker, storedArtifact("ELF binary"))

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
		boundaryExecutor{},
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

	want := "/sandbox/Checker /sandbox/input.txt /sandbox/output.txt /sandbox/answer.txt"
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

// The same path without a custom checker must not reach the light pool at all,
// because token comparison still runs in the worker.
func TestJudgeSubmission_NoCustomChecker_ClaimsNoLightPoolContainer(t *testing.T) {
	docker := &mockDockerExecClient{
		copyFromContainerFn: func(context.Context, string, client.CopyFromContainerOptions) (client.CopyFromContainerResult, error) {
			return client.CopyFromContainerResult{Content: outputTar([]byte("3"))}, nil
		},
	}
	checker, poolDocker := newTestOutputChecker(t, docker, storedArtifact("ELF binary"))

	sub := pendingBoundarySubmission()
	uc := appjudge.NewJudgeSubmissionUseCase(
		&boundarySubmissionUpdater{sub: sub},
		boundaryDownloader{},
		&boundaryProblemProvider{limits: appjudge.ProblemLimits{TimeLimitMs: 1000, MemoryKb: 262144}},
		boundaryTestCaseProvider{},
		boundaryExecutor{},
		checker,
		boundaryTxManager{},
		appjudge.RetryConfig{MaxAttempts: 1},
	)

	if err := uc.Execute(context.Background(), appjudge.JudgeSubmissionInput{SubmissionID: boundarySubmissionID}); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	if got := poolDocker.idCounter.Load(); got != 0 {
		t.Errorf("expected no light pool container, the pool created %d", got)
	}
	if got := sub.Status().String(); got != "ACCEPTED" {
		t.Errorf("verdict: got %q, want ACCEPTED — the tokens match", got)
	}
}
