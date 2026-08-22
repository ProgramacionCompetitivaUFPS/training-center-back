package judge

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/moby/moby/client"
	judgepool "github.com/training-judge-center/backend/internal/adapter/judge/pool"
)

// newTestArtifactSession builds the shared session half over its own pool, which
// is returned so a test can tell a released container from a discarded one.
func newTestArtifactSession(t *testing.T, docker *mockDockerExecClient) (*artifactSession, *mockPoolDockerClient) {
	t.Helper()
	p, poolDocker := newTestPool(t)
	c, err := p.Claim(context.Background(), testLang, judgepool.LanguageCeiling)
	if err != nil {
		t.Fatalf("pool.Claim: %v", err)
	}
	s := &artifactSession{
		container: c,
		pool:      p,
		docker:    docker,
		role:      "Checker",
		runCmd:    "/sandbox/Checker",
	}
	return s, poolDocker
}

// The in-container timeout and the worker's safety net have to stay coupled, so
// run owns both instead of leaving the wrapper to each caller.
func TestArtifactSession_Run_WrapsTheCommandInTheInContainerTimeout(t *testing.T) {
	var cmds [][]string
	docker := &mockDockerExecClient{}
	recordExecs(docker, &cmds)
	s, _ := newTestArtifactSession(t, docker)

	if _, _, err := s.run(context.Background(), "/sandbox/Checker a b c"); err != nil {
		t.Fatalf("run: %v", err)
	}

	if len(cmds) != 1 || len(cmds[0]) != 3 || cmds[0][0] != "sh" || cmds[0][1] != "-c" {
		t.Fatalf("expected one command through sh -c, got: %v", cmds)
	}
	want := "timeout --kill-after=1s 30s /sandbox/Checker a b c"
	if cmds[0][2] != want {
		t.Errorf("command: got %q, want %q", cmds[0][2], want)
	}
}

// Reaching the safety net means the in-container timeout did not fire, so the
// container is in an unknown state and must not go back to the pool.
func TestArtifactSession_Run_SafetyNetDiscardsTheContainer(t *testing.T) {
	docker := &mockDockerExecClient{
		execAttachFn: func(ctx context.Context, _ string, _ client.ExecAttachOptions) (client.ExecAttachResult, error) {
			return blockingAttach(ctx), nil
		},
	}
	s, poolDocker := newTestArtifactSession(t, docker)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	if _, _, err := s.run(ctx, "/sandbox/Checker"); err == nil {
		t.Fatal("expected an error from the safety net, got nil")
	}
	if s.container != nil {
		t.Error("the container must be nil after being discarded")
	}
	if _, err := s.pool.Claim(context.Background(), testLang, judgepool.LanguageCeiling); err != nil {
		t.Fatalf("claim after discard: %v", err)
	}
	if got := poolDocker.idCounter.Load(); got != 2 {
		t.Errorf("expected a fresh container after the discard, the pool created %d", got)
	}
}

// The next claimer runs another problem's artifact, which must not find this
// one lying in /sandbox.
func TestArtifactSession_Close_WipesTheSandboxAndReturnsTheContainer(t *testing.T) {
	var cmds [][]string
	docker := &mockDockerExecClient{}
	recordExecs(docker, &cmds)
	s, poolDocker := newTestArtifactSession(t, docker)

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
	if _, err := s.pool.Claim(context.Background(), testLang, judgepool.LanguageCeiling); err != nil {
		t.Fatalf("claim after close: %v", err)
	}
	if got := poolDocker.idCounter.Load(); got != 1 {
		t.Errorf("expected the container to be reused, but the pool created %d", got)
	}
}

func TestArtifactSession_Close_DiscardsTheContainerWhenCleanupFails(t *testing.T) {
	docker := &mockDockerExecClient{
		execCreateFn: func(_ context.Context, _ string, _ client.ExecCreateOptions) (client.ExecCreateResult, error) {
			return client.ExecCreateResult{}, errors.New("daemon unreachable")
		},
	}
	s, poolDocker := newTestArtifactSession(t, docker)

	if err := s.Close(context.Background()); err != nil {
		t.Fatalf("Close: %v", err)
	}

	if _, err := s.pool.Claim(context.Background(), testLang, judgepool.LanguageCeiling); err != nil {
		t.Fatalf("claim after close: %v", err)
	}
	if got := poolDocker.idCounter.Load(); got != 2 {
		t.Errorf("expected the dirty container to be discarded and a fresh one created, the pool created %d", got)
	}
}

// A killed artifact is our memory limit, not a verdict: reporting it as a
// non-zero exit would make the checker blame the contestant's output, and the
// validator blame the setter's test case. Caught in run so both sessions get it.
func TestArtifactSession_Run_KilledArtifactIsAnErrorNotAnExitCode(t *testing.T) {
	docker := &mockDockerExecClient{
		execInspectFn: func(_ context.Context, _ string, _ client.ExecInspectOptions) (client.ExecInspectResult, error) {
			return client.ExecInspectResult{ExitCode: 137}, nil
		},
	}
	s, _ := newTestArtifactSession(t, docker)

	exitCode, _, err := s.run(context.Background(), "/sandbox/Checker")
	if err == nil {
		t.Fatal("expected an error for a killed artifact, got nil")
	}
	if exitCode != 0 {
		t.Errorf("exit code = %d, want 0 — a killed artifact must not reach the caller as a verdict", exitCode)
	}
}

// Any other non-zero exit is the artifact doing its job, and has to keep
// travelling back as a result.
func TestArtifactSession_Run_OtherNonZeroExitsStayResults(t *testing.T) {
	for _, code := range []int{1, 2, 3, 42} {
		docker := &mockDockerExecClient{
			execInspectFn: func(_ context.Context, _ string, _ client.ExecInspectOptions) (client.ExecInspectResult, error) {
				return client.ExecInspectResult{ExitCode: code}, nil
			},
		}
		s, _ := newTestArtifactSession(t, docker)

		got, _, err := s.run(context.Background(), "/sandbox/Checker")
		if err != nil {
			t.Fatalf("exit %d: unexpected error: %v", code, err)
		}
		if got != code {
			t.Errorf("exit code = %d, want %d", got, code)
		}
	}
}
