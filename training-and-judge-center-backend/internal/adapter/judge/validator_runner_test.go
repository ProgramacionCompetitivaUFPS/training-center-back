package judge

import (
	"archive/tar"
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"cloud.google.com/go/storage"
	"github.com/moby/moby/client"
	judgepool "github.com/training-judge-center/backend/internal/adapter/judge/pool"
	appjudge "github.com/training-judge-center/backend/internal/application/judge"
	"github.com/training-judge-center/backend/internal/domain/submission"
	"github.com/training-judge-center/backend/pkg/apperror"
)

const testValidatorKey = "problems/abc/validator/compiled"

func testValidatorCfg() ArtifactConfig {
	return ArtifactConfig{
		Languages: map[string]ArtifactLanguageConfig{
			testLang: {
				SourcePath:   "/sandbox/{name}.cpp",
				CompileCmd:   "g++ -o /sandbox/{name} /sandbox/{name}.cpp",
				ArtifactPath: "/sandbox/{name}",
				RunCmd:       "/sandbox/{name}",
			},
		},
	}
}

func storedArtifact(content string) *mockGCSReader {
	return &mockGCSReader{
		readObjectFn: func(_ context.Context, _ string) (io.ReadCloser, error) {
			return io.NopCloser(strings.NewReader(content)), nil
		},
	}
}

// newTestValidatorRunner builds the adapter over its own pool, returned so a
// test can tell a released container from a discarded one.
func newTestValidatorRunner(t *testing.T, docker *mockDockerExecClient, reader gcsReader) (*ValidatorRunner, *mockPoolDockerClient) {
	t.Helper()
	p, poolDocker := newTestPoolForExecutor(t)
	return &ValidatorRunner{pool: p, docker: docker, reader: reader, cfg: testValidatorCfg()}, poolDocker
}

func beginTestValidating(t *testing.T, r *ValidatorRunner) appjudge.ValidatorSession {
	t.Helper()
	s, err := r.BeginValidating(context.Background(), testValidatorKey, submission.RestoreLanguage(testLang))
	if err != nil {
		t.Fatalf("BeginValidating: %v", err)
	}
	return s
}

// firstTarEntryMode is firstTarEntry plus the mode, which is what decides
// whether the sandbox can execute the artifact at all.
func firstTarEntryMode(t *testing.T, r io.Reader) (string, int64, []byte) {
	t.Helper()
	tr := tar.NewReader(r)
	hdr, err := tr.Next()
	if err != nil {
		t.Fatalf("could not read the tar sent to the container: %v", err)
	}
	data, _ := io.ReadAll(tr)
	return hdr.Name, hdr.Mode, data
}

// A C++ artifact is an ELF binary the sandbox runs directly, so injecting it
// without the executable bit makes every validation fail with exit 126.
func TestValidatorRunner_BeginValidating_InjectsTheArtifactExecutable(t *testing.T) {
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
	r, _ := newTestValidatorRunner(t, docker, storedArtifact("ELF binary"))

	beginTestValidating(t, r)

	if gotDest != "/sandbox" {
		t.Errorf("destination: got %q, want /sandbox", gotDest)
	}
	if gotName != "Validator" {
		t.Errorf("artifact file: got %q, want Validator", gotName)
	}
	if gotMode != modeExecutable {
		t.Errorf("artifact mode: got %#o, want %#o", gotMode, modeExecutable)
	}
	if string(gotContent) != "ELF binary" {
		t.Errorf("artifact bytes: got %q", gotContent)
	}
}

func TestValidatorSession_Validate_RunsTheArtifactOnTheInput(t *testing.T) {
	var cmds [][]string
	var gotInput []byte
	docker := &mockDockerExecClient{
		copyToContainerFn: func(_ context.Context, _ string, opts client.CopyToContainerOptions) (client.CopyToContainerResult, error) {
			name, _, content := firstTarEntryMode(t, opts.Content)
			if name == "input.txt" {
				gotInput = content
			}
			return client.CopyToContainerResult{}, nil
		},
	}
	recordExecs(docker, &cmds)
	r, _ := newTestValidatorRunner(t, docker, storedArtifact("ELF binary"))
	s := beginTestValidating(t, r)

	result, err := s.Validate(context.Background(), []byte("5 7"))

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Accepted {
		t.Errorf("expected the input to be accepted, got %+v", result)
	}
	if string(gotInput) != "5 7" {
		t.Errorf("input: got %q, want %q", gotInput, "5 7")
	}
	if len(cmds) != 1 || len(cmds[0]) != 3 || cmds[0][0] != "sh" || cmds[0][1] != "-c" {
		t.Fatalf("expected one command through sh -c, got: %v", cmds)
	}
	want := "timeout --kill-after=1s 30s /sandbox/Validator < /sandbox/input.txt"
	if cmds[0][2] != want {
		t.Errorf("run command: got %q, want %q", cmds[0][2], want)
	}
}

// A rejected test case is the validator doing its job, so it travels back as a
// result carrying its stderr, never as an error.
func TestValidatorSession_Validate_RejectionIsAResultNotAnError(t *testing.T) {
	docker := &mockDockerExecClient{
		execInspectFn: func(_ context.Context, _ string, _ client.ExecInspectOptions) (client.ExecInspectResult, error) {
			return client.ExecInspectResult{ExitCode: 1}, nil
		},
		execAttachFn: func(_ context.Context, _ string, _ client.ExecAttachOptions) (client.ExecAttachResult, error) {
			return fakeAttach(stdcopyFrame(2, []byte("FAIL n is out of range\n"))), nil
		},
	}
	r, _ := newTestValidatorRunner(t, docker, storedArtifact("ELF binary"))
	s := beginTestValidating(t, r)

	result, err := s.Validate(context.Background(), []byte("999999"))

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if result.Accepted {
		t.Fatal("expected the input to be rejected")
	}
	if result.Message != "FAIL n is out of range" {
		t.Errorf("message: got %q, want the validator's stderr", result.Message)
	}
}

// The session exists so that one container and one download serve every input.
func TestValidatorSession_Validate_ReusesOneContainerForEveryInput(t *testing.T) {
	downloads := 0
	reader := &mockGCSReader{
		readObjectFn: func(_ context.Context, _ string) (io.ReadCloser, error) {
			downloads++
			return io.NopCloser(strings.NewReader("ELF binary")), nil
		},
	}
	docker := &mockDockerExecClient{}
	r, poolDocker := newTestValidatorRunner(t, docker, reader)
	s := beginTestValidating(t, r)

	for range 3 {
		if _, err := s.Validate(context.Background(), []byte("1")); err != nil {
			t.Fatalf("Validate: %v", err)
		}
	}

	if downloads != 1 {
		t.Errorf("expected the artifact to be downloaded once, got %d", downloads)
	}
	if got := poolDocker.idCounter.Load(); got != 1 {
		t.Errorf("expected one container for the whole session, the pool created %d", got)
	}
}

// The next claimer runs another problem's checker, which must not find this
// validator lying in /sandbox.
func TestValidatorSession_Close_WipesTheSandboxAndReturnsTheContainer(t *testing.T) {
	var cmds [][]string
	docker := &mockDockerExecClient{}
	recordExecs(docker, &cmds)
	r, poolDocker := newTestValidatorRunner(t, docker, storedArtifact("ELF binary"))
	s := beginTestValidating(t, r)

	if err := s.Close(context.Background()); err != nil {
		t.Fatalf("Close: %v", err)
	}

	var wiped bool
	for _, cmd := range cmds {
		if len(cmd) == 3 && strings.Contains(cmd[2], "rm -rf /sandbox/*") {
			wiped = true
		}
	}
	if !wiped {
		t.Errorf("expected the sandbox to be wiped, commands were: %v", cmds)
	}
	if _, err := r.pool.Claim(context.Background(), testLang); err != nil {
		t.Fatalf("claim after close: %v", err)
	}
	if got := poolDocker.idCounter.Load(); got != 1 {
		t.Errorf("expected the container to be reused, but the pool created %d", got)
	}
}

func TestValidatorSession_Close_DiscardsTheContainerWhenCleanupFails(t *testing.T) {
	execs := 0
	docker := &mockDockerExecClient{}
	docker.execCreateFn = func(_ context.Context, _ string, _ client.ExecCreateOptions) (client.ExecCreateResult, error) {
		execs++
		if execs > 1 {
			return client.ExecCreateResult{}, errors.New("daemon unreachable")
		}
		return client.ExecCreateResult{ID: "exec-1"}, nil
	}
	r, poolDocker := newTestValidatorRunner(t, docker, storedArtifact("ELF binary"))
	s := beginTestValidating(t, r)

	if _, err := s.Validate(context.Background(), []byte("1")); err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if err := s.Close(context.Background()); err != nil {
		t.Fatalf("Close: %v", err)
	}

	if _, err := r.pool.Claim(context.Background(), testLang); err != nil {
		t.Fatalf("claim after close: %v", err)
	}
	if got := poolDocker.idCounter.Load(); got != 2 {
		t.Errorf("expected the dirty container to be discarded and a fresh one created, the pool created %d", got)
	}
}

// Downloading before claiming is what keeps the failure path free of a
// container that would have to be handed back.
func TestValidatorRunner_BeginValidating_MissingArtifactClaimsNoContainer(t *testing.T) {
	reader := &mockGCSReader{
		readObjectFn: func(_ context.Context, _ string) (io.ReadCloser, error) {
			return nil, storage.ErrObjectNotExist
		},
	}
	docker := &mockDockerExecClient{}
	r, poolDocker := newTestValidatorRunner(t, docker, reader)

	_, err := r.BeginValidating(context.Background(), testValidatorKey, submission.RestoreLanguage(testLang))

	assertAppErrorKind(t, err, apperror.KindInternal)
	if poolDocker.idCounter.Load() != 0 {
		t.Error("expected no container to be claimed")
	}
}

// java17 is a language the pool can hand out but the artifact config says
// nothing about, so the rejection has to come from the adapter's own guard.
func TestValidatorRunner_BeginValidating_RejectsALanguageWithNoArtifactConfig(t *testing.T) {
	poolCfg := testPoolCfg()
	poolCfg.Languages["java17"] = judgepool.LanguageConfig{Image: "judge:java17", MemoryBytes: testMemBytes}
	poolMock := &mockPoolDockerClient{}
	p := judgepool.NewPool(poolCfg, poolMock)
	p.Start()
	t.Cleanup(p.Stop)

	r := &ValidatorRunner{pool: p, docker: &mockDockerExecClient{}, reader: storedArtifact("jar"), cfg: testValidatorCfg()}

	_, err := r.BeginValidating(context.Background(), testValidatorKey, submission.RestoreLanguage("java17"))

	assertAppErrorKind(t, err, apperror.KindInternal)
	if poolMock.idCounter.Load() != 0 {
		t.Error("expected no container to be claimed")
	}
}
