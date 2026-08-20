package judge

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/moby/moby/api/pkg/stdcopy"
	"github.com/moby/moby/client"

	"github.com/training-judge-center/backend/internal/adapter/judge/pool"
	appjudge "github.com/training-judge-center/backend/internal/application/judge"
	"github.com/training-judge-center/backend/pkg/apperror"
)

var _ appjudge.ValidatorSession = (*ValidatorSession)(nil)

const (
	// artifactRunSecs bounds the artifact inside the container; artifactRunGrace
	// is the extra time the worker waits before assuming the daemon itself is
	// stuck and the container has to go.
	artifactRunSecs  = 30
	artifactRunGrace = 5 * time.Second

	validatorInputPath = "/sandbox/input.txt"
)

type ValidatorSession struct {
	container *pool.Container // nil once discarded
	pool      *pool.Pool
	docker    dockerExecClient
	runCmd    string // the artifact's run command, role name already substituted
}

// Validate feeds one input to the validator on stdin. A non-zero exit is the
// validator rejecting the test case, which is a result, not a failure of ours.
func (s *ValidatorSession) Validate(ctx context.Context, input []byte) (appjudge.ValidatorRunResult, error) {
	if s.container == nil {
		slog.ErrorContext(ctx, "validator_session: validate called on a discarded session")
		return appjudge.ValidatorRunResult{}, apperror.NewInternal()
	}

	if _, err := s.docker.CopyToContainer(ctx, s.container.ID(), client.CopyToContainerOptions{
		DestinationPath: "/sandbox",
		Content:         buildTar("input.txt", input, modeSource),
	}); err != nil {
		slog.ErrorContext(ctx, "validator_session: copy input failed", "container_id", s.container.ID(), "error", err)
		return appjudge.ValidatorRunResult{}, apperror.NewInternal()
	}

	cmd := fmt.Sprintf("timeout --kill-after=1s %ds %s < %s", artifactRunSecs, s.runCmd, validatorInputPath)
	safetyCtx, cancel := context.WithTimeout(ctx, artifactRunSecs*time.Second+artifactRunGrace)
	defer cancel()

	execRes, err := s.docker.ExecCreate(safetyCtx, s.container.ID(), client.ExecCreateOptions{
		Cmd:          []string{"sh", "-c", cmd},
		AttachStdout: true,
		AttachStderr: true,
	})
	if err != nil {
		slog.ErrorContext(ctx, "validator_session: exec create failed", "container_id", s.container.ID(), "error", err)
		return appjudge.ValidatorRunResult{}, apperror.NewInternal()
	}

	att, err := s.docker.ExecAttach(safetyCtx, execRes.ID, client.ExecAttachOptions{})
	if err != nil {
		slog.ErrorContext(ctx, "validator_session: exec attach failed", "container_id", s.container.ID(), "error", err)
		return appjudge.ValidatorRunResult{}, apperror.NewInternal()
	}
	stop := context.AfterFunc(safetyCtx, func() { att.Conn.Close() })
	defer stop()

	var outBuf, errBuf bytes.Buffer
	_, _ = stdcopy.StdCopy(&outBuf, &errBuf, att.Reader)

	// The in-container timeout should have ended this long ago, so reaching the
	// safety net means the container is in an unknown state.
	if safetyCtx.Err() != nil {
		slog.ErrorContext(ctx, "validator_session: safety net activated", "container_id", s.container.ID())
		s.pool.Discard(ctx, s.container)
		s.container = nil
		return appjudge.ValidatorRunResult{}, apperror.NewInternal()
	}
	att.Conn.Close()

	inspectRes, err := s.docker.ExecInspect(ctx, execRes.ID, client.ExecInspectOptions{})
	if err != nil {
		slog.ErrorContext(ctx, "validator_session: exec inspect failed", "container_id", s.container.ID(), "error", err)
		return appjudge.ValidatorRunResult{}, apperror.NewInternal()
	}
	if inspectRes.ExitCode != 0 {
		return appjudge.ValidatorRunResult{Accepted: false, Message: strings.TrimSpace(errBuf.String())}, nil
	}
	return appjudge.ValidatorRunResult{Accepted: true}, nil
}

// Close wipes the sandbox before the container goes back to the pool: the next
// claimer runs another problem's checker, which must not find this validator.
// A container that cannot be wiped is destroyed instead of handed back.
func (s *ValidatorSession) Close(ctx context.Context) error {
	if s.container == nil {
		return nil
	}
	cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), cleanupTimeout)
	defer cancel()

	if err := runAndWait(cleanupCtx, s.docker, s.container.ID(), []string{"sh", "-c", "rm -rf /sandbox/*"}); err != nil {
		slog.ErrorContext(ctx, "validator_session: sandbox cleanup failed, discarding container", "container_id", s.container.ID(), "error", err)
		s.pool.Discard(cleanupCtx, s.container)
		s.container = nil
		return nil
	}
	s.pool.Release(s.container)
	s.container = nil
	return nil
}
