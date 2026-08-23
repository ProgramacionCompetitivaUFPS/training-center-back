package judge

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/moby/moby/client"
	judgepool "github.com/training-judge-center/backend/internal/adapter/judge/pool"
	appjudge "github.com/training-judge-center/backend/internal/application/judge"
	"github.com/training-judge-center/backend/internal/domain/submission"
)

func testExecCfg() ExecutorConfig {
	return ExecutorConfig{
		Languages: map[string]LanguageExecConfig{
			testLang: {
				CompileCmd:   "g++ -std=c++20 -o /sandbox/solution /sandbox/solution.cpp",
				RunCmd:       "/sandbox/solution",
				Extension:    "cpp",
				MemoryFactor: 1.0,
			},
		},
	}
}

func newTestSession(t *testing.T, docker *mockDockerExecClient) (*Session, *judgepool.Pool) {
	t.Helper()
	p, _ := newTestPool(t)
	c, err := p.Claim(context.Background(), testLang, judgepool.LanguageCeiling)
	if err != nil {
		t.Fatalf("pool.Claim: %v", err)
	}
	s := &Session{
		container:  c,
		pool:       p,
		docker:     docker,
		langCfg:    testExecCfg().Languages[testLang],
		judgingDir: newTestJudgingDir(t),
	}
	return s, p
}

// --- Tests ---

// 1. BeginSession with a language absent from ExecutorConfig returns error without claiming.
func TestExecutor_BeginSession_UnknownLanguage(t *testing.T) {
	p, poolMock := newTestPool(t)
	e := NewExecutor(p, &mockDockerExecClient{}, ExecutorConfig{Languages: map[string]LanguageExecConfig{}}, t.TempDir())

	_, err := e.BeginSession(context.Background(), submission.RestoreLanguage("rust"), testProblemMemoryKb, "judging-1")
	if err == nil {
		t.Fatal("expected error for unknown language, got nil")
	}
	if poolMock.idCounter.Load() != 0 {
		t.Errorf("containers created = %d, want 0 — Claim must not run for unknown language", poolMock.idCounter.Load())
	}
}

// 2. BeginSession for a known language claims a container and returns a *Session.
func TestExecutor_BeginSession_Success(t *testing.T) {
	p, _ := newTestPool(t)
	e := NewExecutor(p, &mockDockerExecClient{}, testExecCfg(), t.TempDir())

	sess, err := e.BeginSession(context.Background(), submission.RestoreLanguage(testLang), testProblemMemoryKb, "judging-1")
	if err != nil {
		t.Fatalf("BeginSession: %v", err)
	}
	actual, ok := sess.(*Session)
	if !ok || actual == nil {
		t.Fatal("expected *Session, got nil or wrong type")
	}
	if actual.container == nil {
		t.Error("session container is nil after BeginSession")
	}
	_ = actual.Close(context.Background())
}

// 3. Compile does nothing for interpreted languages (CompileCmd == "").
func TestSession_Compile_SkippedForInterpreted(t *testing.T) {
	mock := &mockDockerExecClient{}
	s, p := newTestSession(t, mock)

	s.langCfg.CompileCmd = ""

	result, err := s.Compile(context.Background(), appjudge.CompileRequest{SourceCode: []byte("print('hi')")})
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	if !result.Success {
		t.Error("expected Success=true for interpreted language")
	}
	if mock.execCreateCnt.Load() != 0 {
		t.Errorf("execCreateCnt = %d, want 0 — no Docker calls for interpreted language", mock.execCreateCnt.Load())
	}
	p.Release(s.container) // bypass Close to avoid counting the cleanup ExecCreate
}

// 4. Compile with exit code 0 returns {Success: true}.
func TestSession_Compile_Success(t *testing.T) {
	mock := &mockDockerExecClient{
		execInspectFn: func(_ context.Context, _ string, _ client.ExecInspectOptions) (client.ExecInspectResult, error) {
			return client.ExecInspectResult{ExitCode: 0}, nil
		},
	}
	s, _ := newTestSession(t, mock)
	defer s.Close(context.Background())

	result, err := s.Compile(context.Background(), appjudge.CompileRequest{SourceCode: []byte("int main(){}")})
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	if !result.Success {
		t.Error("expected Success=true for exit code 0")
	}
}

// 5. Compile with exit code 1 returns {Success: false, Log: compilerOutput}.
func TestSession_Compile_Failure(t *testing.T) {
	errText := "error: expected ';'"
	mock := &mockDockerExecClient{
		execAttachFn: func(_ context.Context, _ string, _ client.ExecAttachOptions) (client.ExecAttachResult, error) {
			return fakeAttach(stdcopyFrame(2, []byte(errText))), nil
		},
		execInspectFn: func(_ context.Context, _ string, _ client.ExecInspectOptions) (client.ExecInspectResult, error) {
			return client.ExecInspectResult{ExitCode: 1}, nil
		},
	}
	s, _ := newTestSession(t, mock)
	defer s.Close(context.Background())

	result, err := s.Compile(context.Background(), appjudge.CompileRequest{SourceCode: []byte("int main(){")})
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	if result.Success {
		t.Error("expected Success=false for exit code 1")
	}
	if result.Log != errText {
		t.Errorf("Log = %q, want %q", result.Log, errText)
	}
}

// 6. Compile log exceeding 10 KB is truncated to maxCompileLogBytes.
func TestSession_Compile_LogTruncated(t *testing.T) {
	bigLog := bytes.Repeat([]byte("x"), 20*1024)
	mock := &mockDockerExecClient{
		execAttachFn: func(_ context.Context, _ string, _ client.ExecAttachOptions) (client.ExecAttachResult, error) {
			return fakeAttach(stdcopyFrame(2, bigLog)), nil
		},
		execInspectFn: func(_ context.Context, _ string, _ client.ExecInspectOptions) (client.ExecInspectResult, error) {
			return client.ExecInspectResult{ExitCode: 1}, nil
		},
	}
	s, _ := newTestSession(t, mock)
	defer s.Close(context.Background())

	result, err := s.Compile(context.Background(), appjudge.CompileRequest{SourceCode: []byte("x")})
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	if len(result.Log) != maxCompileLogBytes {
		t.Errorf("len(Log) = %d, want %d", len(result.Log), maxCompileLogBytes)
	}
}

// 7. RunTestCase with exit code 0 previews what the program left in the volume.
func TestSession_RunTestCase_Accepted(t *testing.T) {
	expected := []byte("42\n")
	mock := &mockDockerExecClient{}
	s, _ := newTestSession(t, mock)
	defer s.Close(context.Background())
	writeContestantOutput(t, s.judgingDir, expected)

	result, err := s.RunTestCase(context.Background(), appjudge.RunRequest{
		Input: []byte("1\n"), TimeLimitMs: 1000,
	})
	if err != nil {
		t.Fatalf("RunTestCase: %v", err)
	}
	if result.ExitCode != 0 {
		t.Errorf("ExitCode = %d, want 0", result.ExitCode)
	}
	if !bytes.Equal(result.OutputPreview, expected) {
		t.Errorf("OutputPreview = %q, want %q", result.OutputPreview, expected)
	}
}

// 8. RunTestCase exit code 124 signals TLE (timeout command killed the process).
func TestSession_RunTestCase_TLE(t *testing.T) {
	mock := &mockDockerExecClient{
		execInspectFn: func(_ context.Context, _ string, _ client.ExecInspectOptions) (client.ExecInspectResult, error) {
			return client.ExecInspectResult{ExitCode: 124}, nil
		},
	}
	s, _ := newTestSession(t, mock)
	defer s.Close(context.Background())

	result, err := s.RunTestCase(context.Background(), appjudge.RunRequest{
		Input: []byte("1\n"), TimeLimitMs: 1000,
	})
	if err != nil {
		t.Fatalf("RunTestCase: %v", err)
	}
	if result.ExitCode != 124 {
		t.Errorf("ExitCode = %d, want 124", result.ExitCode)
	}
}

// 9. RunTestCase exit code 137 signals MLE (OOM kill by cgroup, SIGKILL = 128+9).
func TestSession_RunTestCase_MLE(t *testing.T) {
	mock := &mockDockerExecClient{
		execInspectFn: func(_ context.Context, _ string, _ client.ExecInspectOptions) (client.ExecInspectResult, error) {
			return client.ExecInspectResult{ExitCode: 137}, nil
		},
	}
	s, _ := newTestSession(t, mock)
	defer s.Close(context.Background())

	result, err := s.RunTestCase(context.Background(), appjudge.RunRequest{
		Input: []byte("1\n"), TimeLimitMs: 1000,
	})
	if err != nil {
		t.Fatalf("RunTestCase: %v", err)
	}
	if result.ExitCode != 137 {
		t.Errorf("ExitCode = %d, want 137", result.ExitCode)
	}
}

// 10. The memory a run used comes from what /usr/bin/time left behind, not from
// the container's cgroup: MemoryStats.MaxUsage is a cgroup v1 field that reports
// nothing on v2, and the container is reused across test cases anyway.
func TestSession_RunTestCase_MemoryComesFromTheRunsOwnMeasurement(t *testing.T) {
	s, _ := newTestSession(t, &mockDockerExecClient{})
	defer s.Close(context.Background())
	writeMemoryMeasurement(t, s.judgingDir, "262144\n")

	result, err := s.RunTestCase(context.Background(), appjudge.RunRequest{
		Input: []byte("1\n"), TimeLimitMs: 1000,
	})
	if err != nil {
		t.Fatalf("RunTestCase: %v", err)
	}
	if result.MemoryKb == nil {
		t.Fatal("MemoryKb is nil, want the measurement the run left")
	}
	if *result.MemoryKb != 262144 {
		t.Errorf("MemoryKb = %d, want 262144", *result.MemoryKb)
	}
}

// Nothing measurable is reported as nothing: a zero would tell the contestant
// their solution used no memory at all, which is what the old cgroup field did
// on every single verdict.
func TestSession_RunTestCase_UnreadableMeasurementIsNilNotZero(t *testing.T) {
	tests := []struct {
		name    string
		content string
	}{
		{"the file is empty", ""},
		{"the file holds something that is not a number", "Command exited with non-zero status 124\n103424"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, _ := newTestSession(t, &mockDockerExecClient{})
			defer s.Close(context.Background())
			writeMemoryMeasurement(t, s.judgingDir, tt.content)

			result, err := s.RunTestCase(context.Background(), appjudge.RunRequest{
				Input: []byte("1\n"), TimeLimitMs: 1000,
			})
			if err != nil {
				t.Fatalf("RunTestCase: %v", err)
			}
			if result.MemoryKb != nil {
				t.Errorf("MemoryKb = %d, want nil", *result.MemoryKb)
			}
		})
	}
}

// 11. Safety net activation discards the container and returns an error.
func TestSession_RunTestCase_SafetyNet_Discards(t *testing.T) {
	mock := &mockDockerExecClient{
		execAttachFn: func(ctx context.Context, _ string, _ client.ExecAttachOptions) (client.ExecAttachResult, error) {
			return blockingAttach(ctx), nil
		},
	}
	s, _ := newTestSession(t, mock)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	_, err := s.RunTestCase(ctx, appjudge.RunRequest{
		Input: []byte("1\n"), TimeLimitMs: 1000,
	})
	if err == nil {
		t.Fatal("expected error from safety net, got nil")
	}
	if s.container != nil {
		t.Error("container must be nil after Discard")
	}
}

// 12. Close transitions the container back to idle so a subsequent Claim reuses it.
func TestSession_Close_ReleasesContainer(t *testing.T) {
	poolMock := &mockPoolDockerClient{}
	p := judgepool.NewPool(testPoolCfg(), poolMock)
	p.Start()
	t.Cleanup(p.Stop)

	c, _ := p.Claim(context.Background(), testLang, judgepool.LanguageCeiling)
	s := &Session{container: c, pool: p, docker: &mockDockerExecClient{}, langCfg: testExecCfg().Languages[testLang]}

	_ = s.Close(context.Background())

	c2, err := p.Claim(context.Background(), testLang, judgepool.LanguageCeiling)
	if err != nil {
		t.Fatalf("Claim after Close: %v", err)
	}
	if poolMock.idCounter.Load() != 1 {
		t.Errorf("containers created = %d, want 1 (must reuse the released one)", poolMock.idCounter.Load())
	}
	p.Release(c2)
}

// 13. Close on a session whose container was discarded is a no-op.
func TestSession_Close_AfterDiscard_Noop(t *testing.T) {
	mock := &mockDockerExecClient{}
	s, _ := newTestSession(t, mock)
	s.container = nil // simulate post-safety-net state

	cntBefore := mock.execCreateCnt.Load()
	if err := s.Close(context.Background()); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if mock.execCreateCnt.Load() != cntBefore {
		t.Error("Close must not make Docker calls when container is nil")
	}
}

// The port speaks kilobytes because that is what the problem declares; the pool
// speaks bytes. The conversion is one multiplication, and getting it wrong gives
// a container a thousandth of its limit without anything failing visibly.
func TestExecutor_BeginSession_ProblemLimitReachesThePoolInBytes(t *testing.T) {
	p, poolMock := newTestPool(t)
	e := NewExecutor(p, &mockDockerExecClient{}, testExecCfg(), t.TempDir())

	sess, err := e.BeginSession(context.Background(), submission.RestoreLanguage(testLang), testProblemMemoryKb, "judging-1")
	if err != nil {
		t.Fatalf("BeginSession: %v", err)
	}
	// The judging directory is unwritable until Close unlocks it, so leaving the
	// session open makes t.TempDir cleanup fail for anyone who is not root.
	defer sess.Close(context.Background())

	const want = int64(testProblemMemoryKb) * 1024
	if got := poolMock.lastCreateMemory.Load(); got != want {
		t.Errorf("container created with Memory = %d bytes, want %d (%d KB)", got, want, testProblemMemoryKb)
	}
}

// timeout(1) SIGKILLs a second after its own deadline, so the worker's deadline
// has to sit past that. Under it, a run that dies late — a Java OOM takes
// seconds, not microseconds — would be discarded as broken infrastructure and
// end in SYSTEM_ERROR instead of producing a verdict.
func TestSession_RunTestCase_SafetyNetOutlivesTheInContainerKill(t *testing.T) {
	const timeLimitMs = 1000
	// wallBackstop = max(2, ceil(2*timeLimit)) = 2s, and timeout --kill-after=1s
	// makes the process certainly dead one second later.
	const certainlyDead = 3 * time.Second

	var deadline time.Time
	var haveDeadline bool
	mock := &mockDockerExecClient{
		execCreateFn: func(ctx context.Context, _ string, _ client.ExecCreateOptions) (client.ExecCreateResult, error) {
			deadline, haveDeadline = ctx.Deadline()
			return client.ExecCreateResult{ID: "exec-1"}, nil
		},
	}
	s, _ := newTestSession(t, mock)

	start := time.Now()
	if _, err := s.RunTestCase(context.Background(), appjudge.RunRequest{
		Input: []byte("1\n"), TimeLimitMs: timeLimitMs,
	}); err != nil {
		t.Fatalf("RunTestCase: %v", err)
	}

	if !haveDeadline {
		t.Fatal("the exec call must carry a deadline")
	}
	if got := deadline.Sub(start); got <= certainlyDead {
		t.Errorf("worker deadline is %v after the start, want more than %v — it would give up before the in-container SIGKILL",
			got, certainlyDead)
	}
}

// The checker and validator paths assert their command; the solution path did
// not, so dropping --kill-after went unnoticed. That flag is what guarantees
// the process is dead before the worker's own deadline, and without it a slow
// run turns into a discarded container and SYSTEM_ERROR instead of a verdict.
func TestSession_RunTestCase_WrapsTheCommandInTheInContainerTimeout(t *testing.T) {
	var cmds [][]string
	mock := &mockDockerExecClient{}
	recordExecs(mock, &cmds)
	s, _ := newTestSession(t, mock)

	if _, err := s.RunTestCase(context.Background(), appjudge.RunRequest{
		Input: []byte("1\n"), TimeLimitMs: 1000,
	}); err != nil {
		t.Fatalf("RunTestCase: %v", err)
	}

	if len(cmds) != 1 || len(cmds[0]) != 3 || cmds[0][0] != "sh" || cmds[0][1] != "-c" {
		t.Fatalf("expected one command through sh -c, got: %v", cmds)
	}
	// The write limit comes first so it reaches everything below it; time wraps
	// timeout and not the other way round; and -q keeps the diagnostic line GNU
	// time prints on a non-zero exit out of the measurement file.
	want := fmt.Sprintf("ulimit -c 0; ulimit -f %d; ", outputLimitBlocks) +
		"/usr/bin/time -q -f %M -o " + judgingMemPath(s.judgingDir) +
		" timeout --kill-after=1s 2s " + testExecCfg().Languages[testLang].RunCmd +
		" < " + judgingInputPath(s.judgingDir) + " > " + judgingOutputPath(s.judgingDir) + " 2>/dev/null"
	if cmds[0][2] != want {
		t.Errorf("command:\n got %q\nwant %q", cmds[0][2], want)
	}
}

// A runtime that reserves memory of its own would otherwise charge that reserve
// to the contestant: without the factor a Java solution gets 86% of the limit
// the problem declared, against 99% for a native binary.
func TestExecutor_BeginSession_MemoryFactorBuysBackTheRuntimeReserve(t *testing.T) {
	p, poolMock := newTestPool(t)
	cfg := testExecCfg()
	lc := cfg.Languages[testLang]
	lc.MemoryFactor = 1.5
	cfg.Languages[testLang] = lc
	e := NewExecutor(p, &mockDockerExecClient{}, cfg, t.TempDir())

	sess, err := e.BeginSession(context.Background(), submission.RestoreLanguage(testLang), testProblemMemoryKb, "judging-1")
	if err != nil {
		t.Fatalf("BeginSession: %v", err)
	}
	// The judging directory is unwritable until Close unlocks it, so leaving the
	// session open makes t.TempDir cleanup fail for anyone who is not root.
	defer sess.Close(context.Background())

	const want = int64(float64(testProblemMemoryKb) * 1024 * 1.5)
	if got := poolMock.lastCreateMemory.Load(); got != want {
		t.Errorf("container created with Memory = %d, want %d (the problem's %d KB times 1.5)",
			got, want, testProblemMemoryKb)
	}
}

// The input reaching the sandbox through the filesystem instead of the Docker
// API is the whole point: it used to be copied twice per test case, once into
// each container.
func TestSession_RunTestCase_WritesTheInputToTheVolume(t *testing.T) {
	docker := &mockDockerExecClient{
		copyToContainerFn: func(context.Context, string, client.CopyToContainerOptions) (client.CopyToContainerResult, error) {
			t.Error("the input went through the Docker API; it belongs in the volume")
			return client.CopyToContainerResult{}, nil
		},
	}
	s, _ := newTestSession(t, docker)
	defer s.Close(context.Background())

	if _, err := s.RunTestCase(context.Background(), appjudge.RunRequest{
		Input: []byte("1 2\n"), TimeLimitMs: 1000,
	}); err != nil {
		t.Fatalf("RunTestCase: %v", err)
	}

	got, err := os.ReadFile(judgingInputPath(s.judgingDir))
	if err != nil {
		t.Fatalf("reading the input back: %v", err)
	}
	if string(got) != "1 2\n" {
		t.Errorf("input.txt: got %q, want the test case input", got)
	}
}

// The safety net nils the container out, so cleanup placed after that guard
// would leave one directory per rescued judging in the shared volume forever.
func TestSession_Close_RemovesTheJudgingDirectoryEvenWithoutAContainer(t *testing.T) {
	s, _ := newTestSession(t, &mockDockerExecClient{})
	dir := s.judgingDir
	s.container = nil // what the safety net leaves behind

	if err := s.Close(context.Background()); err != nil {
		t.Fatalf("Close: %v", err)
	}

	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Errorf("the judging directory survived Close: %v", err)
	}
}

// The exit code cannot carry this: the three runtimes answer the signal that
// stops an oversized write with 153, 1 and 0. What decides is the file size,
// which works because the write limit sits one block above the reported one.
func TestSession_RunTestCase_OutputLimitIsDecidedByTheFileSize(t *testing.T) {
	tests := []struct {
		name        string
		outputBytes int
		want        bool
	}{
		{"well under the limit", 1024, false},
		{"one byte under the limit", maxOutputBytes - 1, false},
		{"exactly the limit is still legal", maxOutputBytes, false},
		{"one byte over", maxOutputBytes + 1, true},
		{"cut off by the kernel at the write limit", outputLimitBlocks * 512, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, _ := newTestSession(t, &mockDockerExecClient{})
			defer s.Close(context.Background())
			writeContestantOutput(t, s.judgingDir, make([]byte, tt.outputBytes))

			result, err := s.RunTestCase(context.Background(), appjudge.RunRequest{
				Input: []byte("1\n"), TimeLimitMs: 1000,
			})
			if err != nil {
				t.Fatalf("RunTestCase: %v", err)
			}
			if result.OutputLimitExceeded != tt.want {
				t.Errorf("OutputLimitExceeded = %v for %d bytes, want %v",
					result.OutputLimitExceeded, tt.outputBytes, tt.want)
			}
		})
	}
}

// The write limit has to land above the reported one, or an output of exactly
// the limit would be cut and then read as having gone over it.
func TestOutputLimitBlocks_LeaveRoomAboveTheReportedLimit(t *testing.T) {
	if got := outputLimitBlocks * 512; got <= maxOutputBytes {
		t.Errorf("the kernel would cut at %d, at or below the %d we report", got, maxOutputBytes)
	}
}
