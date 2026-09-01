package judge

import (
	"archive/tar"
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/moby/moby/client"
	judgepool "github.com/training-judge-center/backend/internal/adapter/judge/pool"
	appjudge "github.com/training-judge-center/backend/internal/application/judge"
	"github.com/training-judge-center/backend/internal/domain/submission"
	"github.com/training-judge-center/backend/pkg/apperror"
)

func newTestArtifactCompiler(t *testing.T, docker *mockDockerExecClient) *ArtifactCompiler {
	t.Helper()
	p, _ := newTestPool(t)
	return NewArtifactCompiler(p, docker, testArtifactCfg())
}

// artifactTar mimics CopyFromContainer's response for the compiled artifact.
func artifactTar(content []byte) io.ReadCloser {
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	_ = tw.WriteHeader(&tar.Header{Name: "artifact", Mode: 0755, Size: int64(len(content))})
	_, _ = tw.Write(content)
	_ = tw.Close()
	return io.NopCloser(&buf)
}

func TestArtifactCompiler_Compile_SubstitutesTheRoleEverywhere(t *testing.T) {
	tests := []struct {
		name         string
		role         appjudge.ArtifactRole
		wantSource   string
		wantCompile  string
		wantArtifact string
	}{
		{
			name:         "checker",
			role:         appjudge.NewArtifactRoleChecker(),
			wantSource:   "Checker.cpp",
			wantCompile:  "g++ -o /sandbox/Checker /sandbox/Checker.cpp",
			wantArtifact: "/sandbox/Checker",
		},
		{
			name:         "validator",
			role:         appjudge.NewArtifactRoleValidator(),
			wantSource:   "Validator.cpp",
			wantCompile:  "g++ -o /sandbox/Validator /sandbox/Validator.cpp",
			wantArtifact: "/sandbox/Validator",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var cmds [][]string
			var gotDest, gotSourceName, gotArtifactPath string
			var gotSource []byte
			docker := &mockDockerExecClient{
				copyToContainerFn: func(_ context.Context, _ string, opts client.CopyToContainerOptions) (client.CopyToContainerResult, error) {
					gotDest = opts.DestinationPath
					gotSourceName, gotSource = firstTarEntry(t, opts.Content)
					return client.CopyToContainerResult{}, nil
				},
				copyFromContainerFn: func(_ context.Context, _ string, opts client.CopyFromContainerOptions) (client.CopyFromContainerResult, error) {
					gotArtifactPath = opts.SourcePath
					return client.CopyFromContainerResult{Content: artifactTar([]byte("ELF binary"))}, nil
				},
			}
			recordExecs(docker, &cmds)
			c := newTestArtifactCompiler(t, docker)

			result, err := c.Compile(context.Background(), appjudge.CompileArtifactRequest{
				Role:       tt.role,
				Language:   submission.RestoreLanguage(testLang),
				SourceCode: []byte("int main() { return 0; }"),
			})

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !result.Success {
				t.Fatalf("expected Success=true, got Log: %s", result.Log)
			}
			if string(result.Artifact) != "ELF binary" {
				t.Errorf("artifact: got %q, want %q", result.Artifact, "ELF binary")
			}
			if gotDest != "/sandbox" {
				t.Errorf("destination: got %q, want %q", gotDest, "/sandbox")
			}
			if gotSourceName != tt.wantSource {
				t.Errorf("source file: got %q, want %q", gotSourceName, tt.wantSource)
			}
			if string(gotSource) != "int main() { return 0; }" {
				t.Errorf("source bytes: got %q", gotSource)
			}
			if gotArtifactPath != tt.wantArtifact {
				t.Errorf("artifact path: got %q, want %q", gotArtifactPath, tt.wantArtifact)
			}
			if len(cmds) == 0 || len(cmds[0]) != 3 || cmds[0][0] != "sh" || cmds[0][1] != "-c" {
				t.Fatalf("expected the compile command to run through sh -c, got: %v", cmds)
			}
			if cmds[0][2] != tt.wantCompile {
				t.Errorf("compile command: got %q, want %q", cmds[0][2], tt.wantCompile)
			}
		})
	}
}

// A checker that does not build is the problem setter's problem, so it comes
// back as a result carrying the compiler's output, never as an error.
func TestArtifactCompiler_Compile_FailedCompilationIsAResultNotAnError(t *testing.T) {
	var copiedFrom bool
	docker := &mockDockerExecClient{
		execInspectFn: func(_ context.Context, id string, _ client.ExecInspectOptions) (client.ExecInspectResult, error) {
			if id == "exec-1" {
				return client.ExecInspectResult{ExitCode: 1}, nil
			}
			return client.ExecInspectResult{ExitCode: 0}, nil
		},
		execAttachFn: func(_ context.Context, id string, _ client.ExecAttachOptions) (client.ExecAttachResult, error) {
			if id == "exec-1" {
				return fakeAttach(stdcopyFrame(2, []byte("error: expected ';' before '}' token"))), nil
			}
			return fakeAttach(nil), nil
		},
		copyFromContainerFn: func(_ context.Context, _ string, _ client.CopyFromContainerOptions) (client.CopyFromContainerResult, error) {
			copiedFrom = true
			return client.CopyFromContainerResult{Content: artifactTar(nil)}, nil
		},
	}
	var cmds [][]string
	recordExecs(docker, &cmds)
	c := newTestArtifactCompiler(t, docker)

	result, err := c.Compile(context.Background(), appjudge.CompileArtifactRequest{
		Role:       appjudge.NewArtifactRoleChecker(),
		Language:   submission.RestoreLanguage(testLang),
		SourceCode: []byte("int main() { }"),
	})

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if result.Success {
		t.Fatal("expected Success=false")
	}
	if !strings.Contains(result.Log, "expected ';'") {
		t.Errorf("expected the compiler output in the log, got: %q", result.Log)
	}
	if result.Artifact != nil {
		t.Error("expected no artifact on a failed compile")
	}
	if copiedFrom {
		t.Error("expected no artifact extraction after a failed compile")
	}
}

// The container goes back to the pool clean: contestant code claiming it next
// must not find the checker's source lying in /sandbox.
func TestArtifactCompiler_Compile_WipesTheSandboxAndReturnsTheContainer(t *testing.T) {
	docker := &mockDockerExecClient{
		copyFromContainerFn: func(_ context.Context, _ string, _ client.CopyFromContainerOptions) (client.CopyFromContainerResult, error) {
			return client.CopyFromContainerResult{Content: artifactTar([]byte("binary"))}, nil
		},
	}
	var cmds [][]string
	recordExecs(docker, &cmds)
	p, poolDocker := newTestPool(t)
	c := NewArtifactCompiler(p, docker, testArtifactCfg())

	if _, err := c.Compile(context.Background(), appjudge.CompileArtifactRequest{
		Role:       appjudge.NewArtifactRoleChecker(),
		Language:   submission.RestoreLanguage(testLang),
		SourceCode: []byte("int main() { return 0; }"),
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
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

	// A container that was released is reused; a discarded one would force the
	// pool to create a second.
	if _, err := p.Claim(context.Background(), testLang, judgepool.LanguageCeiling); err != nil {
		t.Fatalf("claim after compile: %v", err)
	}
	if got := poolDocker.idCounter.Load(); got != 1 {
		t.Errorf("expected the container to be reused, but the pool created %d of them", got)
	}
}

// When the sandbox cannot be wiped, the container is destroyed instead of
// handed to the next claimer with the checker's source still inside it.
func TestArtifactCompiler_Compile_DiscardsTheContainerWhenCleanupFails(t *testing.T) {
	var cmds [][]string
	docker := &mockDockerExecClient{
		copyFromContainerFn: func(_ context.Context, _ string, _ client.CopyFromContainerOptions) (client.CopyFromContainerResult, error) {
			return client.CopyFromContainerResult{Content: artifactTar([]byte("binary"))}, nil
		},
	}
	docker.execCreateFn = func(_ context.Context, _ string, opts client.ExecCreateOptions) (client.ExecCreateResult, error) {
		cmds = append(cmds, opts.Cmd)
		if len(cmds) > 1 {
			return client.ExecCreateResult{}, errors.New("daemon unreachable")
		}
		return client.ExecCreateResult{ID: "exec-1"}, nil
	}
	p, poolDocker := newTestPool(t)
	c := NewArtifactCompiler(p, docker, testArtifactCfg())

	if _, err := c.Compile(context.Background(), appjudge.CompileArtifactRequest{
		Role:       appjudge.NewArtifactRoleChecker(),
		Language:   submission.RestoreLanguage(testLang),
		SourceCode: []byte("int main() { return 0; }"),
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, err := p.Claim(context.Background(), testLang, judgepool.LanguageCeiling); err != nil {
		t.Fatalf("claim after compile: %v", err)
	}
	if got := poolDocker.idCounter.Load(); got != 2 {
		t.Errorf("expected the dirty container to be discarded and a fresh one created, but the pool created %d", got)
	}
}

// The zero value of ArtifactRole is what the type system cannot rule out, so
// the adapter has to: an empty name would write /sandbox/.cpp.
func TestArtifactCompiler_Compile_RejectsARequestWithNoRole(t *testing.T) {
	docker := &mockDockerExecClient{}
	c := newTestArtifactCompiler(t, docker)

	_, err := c.Compile(context.Background(), appjudge.CompileArtifactRequest{
		Language:   submission.RestoreLanguage(testLang),
		SourceCode: []byte("int main() {}"),
	})

	assertAppErrorKind(t, err, apperror.KindInternal)
	if docker.execCreateCnt.Load() != 0 {
		t.Error("expected nothing to run inside a container")
	}
}

// java17 is a language the pool can hand out but the artifact config says
// nothing about — so the rejection has to come from the compiler's own guard,
// not from the pool refusing an unknown language.
func TestArtifactCompiler_Compile_RejectsALanguageWithNoArtifactConfig(t *testing.T) {
	poolCfg := testPoolCfg()
	poolCfg.Languages["java17"] = judgepool.LanguageConfig{Image: "judge:java17", MemoryBytes: testMemBytes}
	poolMock := &mockPoolDockerClient{}
	p := judgepool.NewPool(poolCfg, poolMock)
	p.Start()
	t.Cleanup(p.Stop)

	docker := &mockDockerExecClient{}
	c := NewArtifactCompiler(p, docker, testArtifactCfg())

	_, err := c.Compile(context.Background(), appjudge.CompileArtifactRequest{
		Role:       appjudge.NewArtifactRoleChecker(),
		Language:   submission.RestoreLanguage("java17"),
		SourceCode: []byte("class Checker {}"),
	})

	assertAppErrorKind(t, err, apperror.KindInternal)
	if poolMock.idCounter.Load() != 0 {
		t.Error("expected no container to be claimed")
	}
}

// A command that exits zero without leaving the artifact behind would otherwise
// store an empty object and only break when something tries to run it.
func TestArtifactCompiler_Compile_EmptyArtifactIsAnError(t *testing.T) {
	docker := &mockDockerExecClient{
		copyFromContainerFn: func(_ context.Context, _ string, _ client.CopyFromContainerOptions) (client.CopyFromContainerResult, error) {
			return client.CopyFromContainerResult{Content: artifactTar(nil)}, nil
		},
	}
	c := newTestArtifactCompiler(t, docker)

	_, err := c.Compile(context.Background(), appjudge.CompileArtifactRequest{
		Role:       appjudge.NewArtifactRoleChecker(),
		Language:   submission.RestoreLanguage(testLang),
		SourceCode: []byte("int main() { return 0; }"),
	})

	assertAppErrorKind(t, err, apperror.KindInternal)
}
