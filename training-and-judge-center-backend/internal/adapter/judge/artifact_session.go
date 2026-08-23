package judge

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"path"
	"strings"
	"time"

	"github.com/moby/moby/api/pkg/stdcopy"
	"github.com/moby/moby/client"

	"github.com/training-judge-center/backend/internal/adapter/judge/pool"
	"github.com/training-judge-center/backend/pkg/apperror"
)

const (
	// artifactRunSecs bounds the artifact inside the container; artifactRunGrace
	// is the extra time the worker waits before assuming the daemon itself is
	// stuck and the container has to go.
	artifactRunSecs  = 30
	artifactRunGrace = 5 * time.Second

	// sandboxInputPath is where the validator session puts the input it checks.
	sandboxInputPath = "/sandbox/input.txt"

	// exitCodeKilled is SIGKILL (128 + 9). An artifact reaches it by exceeding
	// the light pool container's memory — from the kernel for C++ and Python,
	// from the JVM's own OnOutOfMemoryError for Java. Either way the artifact
	// died on us, so it is our failure and not a verdict.
	exitCodeKilled = 137
)

// artifactSession is the half a checker session and a validator session share:
// one light pool container running one artifact, and that container's lifecycle.
type artifactSession struct {
	container *pool.Container // nil once discarded
	pool      *pool.Pool
	docker    dockerExecClient
	role      string // Checker or Validator, for logs
	runCmd    string // the artifact's run command, role name already substituted
}

// writeFile puts one file inside the sandbox, overwriting whatever was there.
func (s *artifactSession) writeFile(ctx context.Context, filePath string, content []byte) error {
	if _, err := s.docker.CopyToContainer(ctx, s.container.ID(), client.CopyToContainerOptions{
		DestinationPath: path.Dir(filePath),
		Content:         buildTar(path.Base(filePath), content, modeSource),
	}); err != nil {
		slog.ErrorContext(ctx, "artifact_session: copy file failed", "role", s.role, "container_id", s.container.ID(), "path", filePath, "error", err)
		return apperror.NewInternal()
	}
	return nil
}

// run executes the artifact through sh -c and reports its exit code and stderr.
// It owns the timeout so the in-container limit and the worker's safety net
// cannot drift apart.
func (s *artifactSession) run(ctx context.Context, cmd string) (int, string, error) {
	safetyCtx, cancel := context.WithTimeout(ctx, artifactRunSecs*time.Second+artifactRunGrace)
	defer cancel()

	execRes, err := s.docker.ExecCreate(safetyCtx, s.container.ID(), client.ExecCreateOptions{
		Cmd:          []string{"sh", "-c", fmt.Sprintf("timeout --kill-after=1s %ds %s", artifactRunSecs, cmd)},
		AttachStdout: true,
		AttachStderr: true,
	})
	if err != nil {
		slog.ErrorContext(ctx, "artifact_session: exec create failed", "role", s.role, "container_id", s.container.ID(), "error", err)
		return 0, "", apperror.NewInternal()
	}

	att, err := s.docker.ExecAttach(safetyCtx, execRes.ID, client.ExecAttachOptions{})
	if err != nil {
		slog.ErrorContext(ctx, "artifact_session: exec attach failed", "role", s.role, "container_id", s.container.ID(), "error", err)
		return 0, "", apperror.NewInternal()
	}
	stop := context.AfterFunc(safetyCtx, func() { att.Conn.Close() })
	defer stop()

	var outBuf, errBuf bytes.Buffer
	_, _ = stdcopy.StdCopy(&outBuf, &errBuf, att.Reader)

	// The in-container timeout should have ended this long ago, so reaching the
	// safety net means the container is in an unknown state.
	if safetyCtx.Err() != nil {
		slog.ErrorContext(ctx, "artifact_session: safety net activated", "role", s.role, "container_id", s.container.ID())
		s.pool.Discard(ctx, s.container)
		s.container = nil
		return 0, "", apperror.NewInternal()
	}
	att.Conn.Close()

	inspectRes, err := s.docker.ExecInspect(ctx, execRes.ID, client.ExecInspectOptions{})
	if err != nil {
		slog.ErrorContext(ctx, "artifact_session: exec inspect failed", "role", s.role, "container_id", s.container.ID(), "error", err)
		return 0, "", apperror.NewInternal()
	}
	// Caught here rather than in each session: both read a non-zero exit as the
	// artifact rejecting its input, which for a killed artifact would blame the
	// contestant, or the test case, for our own memory limit.
	if inspectRes.ExitCode == exitCodeKilled {
		slog.ErrorContext(ctx, "artifact_session: artifact was killed, most likely out of memory",
			"role", s.role, "container_id", s.container.ID())
		return 0, "", apperror.NewInternal()
	}
	return inspectRes.ExitCode, strings.TrimSpace(errBuf.String()), nil
}

// Close wipes the sandbox before the container goes back to the pool: the next
// claimer runs another problem's artifact, which must not find this one. A
// container that cannot be wiped is destroyed instead of handed back.
func (s *artifactSession) Close(ctx context.Context) error {
	if s.container == nil {
		return nil
	}
	cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), cleanupTimeout)
	defer cancel()

	if err := runAndWait(cleanupCtx, s.docker, s.container.ID(), []string{"sh", "-c", "rm -rf /sandbox/*"}); err != nil {
		slog.ErrorContext(ctx, "artifact_session: sandbox cleanup failed, discarding container", "role", s.role, "container_id", s.container.ID(), "error", err)
		s.pool.Discard(cleanupCtx, s.container)
		s.container = nil
		return nil
	}
	s.pool.Release(s.container)
	s.container = nil
	return nil
}
