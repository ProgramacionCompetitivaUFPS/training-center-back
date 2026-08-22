package judge

import (
	"bytes"
	"context"
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
				CompileCmd: "g++ -std=c++20 -o /sandbox/solution /sandbox/solution.cpp",
				RunCmd:     "/sandbox/solution",
				Extension:  "cpp",
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
		container: c,
		pool:      p,
		docker:    docker,
		langCfg:   testExecCfg().Languages[testLang],
	}
	return s, p
}

// --- Tests ---

// 1. BeginSession with a language absent from ExecutorConfig returns error without claiming.
func TestExecutor_BeginSession_UnknownLanguage(t *testing.T) {
	p, poolMock := newTestPool(t)
	e := NewExecutor(p, &mockDockerExecClient{}, ExecutorConfig{Languages: map[string]LanguageExecConfig{}})

	_, err := e.BeginSession(context.Background(), submission.RestoreLanguage("rust"), testProblemMemoryKb)
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
	e := NewExecutor(p, &mockDockerExecClient{}, testExecCfg())

	sess, err := e.BeginSession(context.Background(), submission.RestoreLanguage(testLang), testProblemMemoryKb)
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

// 7. RunTestCase with exit code 0 returns the program output.
func TestSession_RunTestCase_Accepted(t *testing.T) {
	expected := []byte("42\n")
	mock := &mockDockerExecClient{
		copyFromContainerFn: func(_ context.Context, _ string, _ client.CopyFromContainerOptions) (client.CopyFromContainerResult, error) {
			return client.CopyFromContainerResult{Content: outputTar(expected)}, nil
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
	if result.ExitCode != 0 {
		t.Errorf("ExitCode = %d, want 0", result.ExitCode)
	}
	if !bytes.Equal(result.Output, expected) {
		t.Errorf("Output = %q, want %q", result.Output, expected)
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

// 10. ContainerStats MaxUsage is reported in KB.
func TestSession_RunTestCase_MemoryKb(t *testing.T) {
	const maxUsage = 256 * 1024 * 1024 // 256 MB → 262144 KB
	mock := &mockDockerExecClient{
		containerStatsFn: func(_ context.Context, _ string, _ client.ContainerStatsOptions) (client.ContainerStatsResult, error) {
			return client.ContainerStatsResult{Body: statsBody(maxUsage)}, nil
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
	if result.MemoryKb != 262144 {
		t.Errorf("MemoryKb = %d, want 262144", result.MemoryKb)
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
	e := NewExecutor(p, &mockDockerExecClient{}, testExecCfg())

	if _, err := e.BeginSession(context.Background(), submission.RestoreLanguage(testLang), testProblemMemoryKb); err != nil {
		t.Fatalf("BeginSession: %v", err)
	}

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
	// wallBackstop = max(2, ceil(2*1000/1000)) = 2s
	want := "timeout --kill-after=1s 2s " + testExecCfg().Languages[testLang].RunCmd +
		" < /sandbox/input.txt > /sandbox/output.txt 2>/dev/null"
	if cmds[0][2] != want {
		t.Errorf("command:\n got %q\nwant %q", cmds[0][2], want)
	}
}
