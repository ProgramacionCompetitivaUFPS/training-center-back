package judge

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"cloud.google.com/go/storage"
	"github.com/moby/moby/client"
	judgepool "github.com/training-judge-center/backend/internal/adapter/judge/pool"
	appjudge "github.com/training-judge-center/backend/internal/application/judge"
	"github.com/training-judge-center/backend/internal/domain/submission"
	"github.com/training-judge-center/backend/pkg/apperror"
)

const (
	testCheckerKey = "problems/abc/checker/compiled"
	testJudgingID  = "judging-1"
)

// newTestOutputChecker builds the adapter over its own pool, returned so a test
// can tell a claimed container from none at all.
func newTestOutputChecker(t *testing.T, docker *mockDockerExecClient, reader gcsReader, judgingRoot string) (*OutputChecker, *mockPoolDockerClient) {
	t.Helper()
	p, poolDocker := newTestPool(t)
	return &OutputChecker{pool: p, docker: docker, reader: reader, cfg: testArtifactCfg(), judgingRoot: judgingRoot}, poolDocker
}

// layOutJudgingDir leaves the directory the way the heavy pool would have: the
// checker only ever finds an existing one.
func layOutJudgingDir(t *testing.T, root, contestantOutput string) string {
	t.Helper()
	dir := filepath.Join(root, testJudgingID)
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatalf("creating the judging directory: %v", err)
	}
	if err := os.WriteFile(judgingOutputPath(dir), []byte(contestantOutput), judgingFileMode); err != nil {
		t.Fatalf("writing the contestant output: %v", err)
	}
	return dir
}

func beginTestChecking(t *testing.T, c *OutputChecker) appjudge.CheckerSession {
	t.Helper()
	s, err := c.BeginChecking(context.Background(), testCheckerKey, submission.RestoreLanguage(testLang), testJudgingID)
	if err != nil {
		t.Fatalf("BeginChecking: %v", err)
	}
	return s
}

// A C++ checker is an ELF binary the sandbox runs directly, so injecting it
// without the executable bit makes every check fail with exit 126.
func TestOutputChecker_BeginChecking_InjectsTheArtifactExecutable(t *testing.T) {
	var gotDest, gotName string
	var gotMode int64
	var gotContent []byte
	docker := &mockDockerExecClient{
		copyToContainerFn: func(_ context.Context, _ string, opts client.CopyToContainerOptions) (client.CopyToContainerResult, error) {
			gotDest = opts.DestinationPath
			gotName, gotMode, gotContent = firstTarEntryMode(t, opts.Content)
			return client.CopyToContainerResult{}, nil
		},
	}
	c, _ := newTestOutputChecker(t, docker, storedArtifact("ELF binary"), t.TempDir())

	beginTestChecking(t, c)

	if gotDest != "/sandbox" {
		t.Errorf("destination: got %q, want /sandbox", gotDest)
	}
	if gotName != "Checker" {
		t.Errorf("artifact file: got %q, want Checker", gotName)
	}
	if gotMode != modeExecutable {
		t.Errorf("artifact mode: got %#o, want %#o", gotMode, modeExecutable)
	}
	if string(gotContent) != "ELF binary" {
		t.Errorf("artifact bytes: got %q", gotContent)
	}
}

// A problem without a custom checker is most of the traffic, and token
// comparison still runs in the worker, so claiming a light pool container for it
// would tie one up per judging for nothing. Step 3 turns this branch into a
// claim of the compare language.
func TestOutputChecker_BeginChecking_NoCheckerPathClaimsNoContainer(t *testing.T) {
	root := t.TempDir()
	c, poolDocker := newTestOutputChecker(t, &mockDockerExecClient{}, storedArtifact("ELF binary"), root)
	layOutJudgingDir(t, root, "42")

	session, err := c.BeginChecking(context.Background(), "", submission.Language{}, testJudgingID)
	if err != nil {
		t.Fatalf("BeginChecking: %v", err)
	}
	defer session.Close(context.Background())

	result, err := session.Check(context.Background(), []byte("42\n"))
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if !result.Accepted {
		t.Error("expected matching tokens to be accepted")
	}
	if got := poolDocker.idCounter.Load(); got != 0 {
		t.Errorf("expected no container to be claimed, the pool created %d", got)
	}
}

// The input and the contestant output come from the judging directory; only the
// jury answer travels through the API, because it is the one file the
// contestant container must never be able to reach. And testlib binds argv[2]
// to the contestant output and argv[3] to the answer: swapped, a malformed
// output makes the checker declare the jury wrong.
func TestCheckerSession_Check_ReadsTheVolumeAndPassesTestlibOrder(t *testing.T) {
	written := map[string][]byte{}
	var cmds [][]string
	docker := &mockDockerExecClient{
		copyToContainerFn: func(_ context.Context, _ string, opts client.CopyToContainerOptions) (client.CopyToContainerResult, error) {
			name, _, content := firstTarEntryMode(t, opts.Content)
			written[name] = content
			return client.CopyToContainerResult{}, nil
		},
	}
	recordExecs(docker, &cmds)
	root := t.TempDir()
	c, _ := newTestOutputChecker(t, docker, storedArtifact("ELF binary"), root)
	dir := layOutJudgingDir(t, root, "3 ")
	s := beginTestChecking(t, c)

	result, err := s.Check(context.Background(), []byte("3"))
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if !result.Accepted {
		t.Errorf("expected exit 0 to be accepted, got %+v", result)
	}

	if string(written["answer.txt"]) != "3" {
		t.Errorf("answer.txt: got %q, want the jury answer", written["answer.txt"])
	}
	for _, name := range []string{"input.txt", "output.txt"} {
		if _, sent := written[name]; sent {
			t.Errorf("%s went through the API; it is supposed to come from the volume", name)
		}
	}

	if len(cmds) != 1 || len(cmds[0]) != 3 || cmds[0][0] != "sh" || cmds[0][1] != "-c" {
		t.Fatalf("expected one command through sh -c, got: %v", cmds)
	}
	want := "timeout --kill-after=1s 30s /sandbox/Checker " +
		judgingInputPath(dir) + " " + judgingOutputPath(dir) + " /sandbox/answer.txt"
	if cmds[0][2] != want {
		t.Errorf("run command: got %q, want %q", cmds[0][2], want)
	}
}

// A rejected output is the checker doing its job, so it travels back as a result
// carrying its stderr, never as an error.
func TestCheckerSession_Check_RejectionIsAResultNotAnError(t *testing.T) {
	docker := &mockDockerExecClient{
		execInspectFn: func(_ context.Context, _ string, _ client.ExecInspectOptions) (client.ExecInspectResult, error) {
			return client.ExecInspectResult{ExitCode: 1}, nil
		},
		execAttachFn: func(_ context.Context, _ string, _ client.ExecAttachOptions) (client.ExecAttachResult, error) {
			return fakeAttach(stdcopyFrame(2, []byte("wrong answer: expected 3, found 4\n"))), nil
		},
	}
	c, _ := newTestOutputChecker(t, docker, storedArtifact("ELF binary"), t.TempDir())
	s := beginTestChecking(t, c)

	result, err := s.Check(context.Background(), []byte("3"))

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if result.Accepted {
		t.Fatal("expected the output to be rejected")
	}
	if result.Message != "wrong answer: expected 3, found 4" {
		t.Errorf("message: got %q, want the checker stderr", result.Message)
	}
}

// The session exists so one container and one download serve every test case,
// instead of the artifact being downloaded once per case as it used to be.
func TestCheckerSession_Check_ReusesOneContainerForEveryTestCase(t *testing.T) {
	downloads := 0
	reader := &mockGCSReader{
		readObjectFn: func(context.Context, string) (io.ReadCloser, error) {
			downloads++
			return io.NopCloser(strings.NewReader("ELF binary")), nil
		},
	}
	c, poolDocker := newTestOutputChecker(t, &mockDockerExecClient{}, reader, t.TempDir())
	s := beginTestChecking(t, c)

	for range 3 {
		if _, err := s.Check(context.Background(), []byte("3")); err != nil {
			t.Fatalf("Check: %v", err)
		}
	}

	if downloads != 1 {
		t.Errorf("expected the artifact to be downloaded once, got %d", downloads)
	}
	if got := poolDocker.idCounter.Load(); got != 1 {
		t.Errorf("expected one container for the whole session, the pool created %d", got)
	}
}

// Downloading before claiming is what keeps the failure path free of a container
// that would have to be handed back.
func TestOutputChecker_BeginChecking_MissingArtifactClaimsNoContainer(t *testing.T) {
	reader := &mockGCSReader{
		readObjectFn: func(context.Context, string) (io.ReadCloser, error) {
			return nil, storage.ErrObjectNotExist
		},
	}
	c, poolDocker := newTestOutputChecker(t, &mockDockerExecClient{}, reader, t.TempDir())

	_, err := c.BeginChecking(context.Background(), testCheckerKey, submission.RestoreLanguage(testLang), testJudgingID)

	assertAppErrorKind(t, err, apperror.KindInternal)
	if poolDocker.idCounter.Load() != 0 {
		t.Error("expected no container to be claimed")
	}
}

// java17 is a language the pool can hand out but the artifact config says
// nothing about, so the rejection has to come from the adapter own guard.
func TestOutputChecker_BeginChecking_RejectsALanguageWithNoArtifactConfig(t *testing.T) {
	poolCfg := testPoolCfg()
	poolCfg.Languages["java17"] = judgepool.LanguageConfig{Image: "judge:java17", MemoryBytes: testMemBytes}
	poolMock := &mockPoolDockerClient{}
	p := judgepool.NewPool(poolCfg, poolMock)
	p.Start()
	t.Cleanup(p.Stop)

	c := &OutputChecker{pool: p, docker: &mockDockerExecClient{}, reader: storedArtifact("jar"), cfg: testArtifactCfg(), judgingRoot: t.TempDir()}

	_, err := c.BeginChecking(context.Background(), testCheckerKey, submission.RestoreLanguage("java17"), testJudgingID)

	assertAppErrorKind(t, err, apperror.KindInternal)
	if poolMock.idCounter.Load() != 0 {
		t.Error("expected no container to be claimed")
	}
}

// The bug this closes: a checker that ran out of memory exited non-zero, which
// Check read as "the contestant output is wrong" — a silent wrong answer for a
// correct solution, caused by our own light pool sizing.
func TestCheckerSession_Check_KilledCheckerIsNotAWrongAnswer(t *testing.T) {
	docker := &mockDockerExecClient{
		execInspectFn: func(_ context.Context, _ string, _ client.ExecInspectOptions) (client.ExecInspectResult, error) {
			return client.ExecInspectResult{ExitCode: 137}, nil
		},
	}
	c, _ := newTestOutputChecker(t, docker, storedArtifact("ELF binary"), t.TempDir())
	s := beginTestChecking(t, c)

	result, err := s.Check(context.Background(), []byte("3"))
	if err == nil {
		t.Fatal("expected an error for a killed checker, got a verdict")
	}
	if result.Accepted {
		t.Error("a killed checker must not accept either")
	}
}
