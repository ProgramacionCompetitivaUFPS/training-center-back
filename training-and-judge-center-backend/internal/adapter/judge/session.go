package judge

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/moby/moby/api/pkg/stdcopy"
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/client"

	"github.com/training-judge-center/backend/internal/adapter/judge/pool"
	appjudge "github.com/training-judge-center/backend/internal/application/judge"
	"github.com/training-judge-center/backend/pkg/apperror"
	"github.com/training-judge-center/backend/pkg/judgelimits"
	"github.com/training-judge-center/backend/pkg/strutil"
)

const (
	maxCompileLogBytes = 10 * 1024
	// maxOutputBytes is the output limit a run is held to, enforced below and
	// reported as a verdict. It is also what a checker has to hold in a light
	// pool container: 64 MiB made every language run out of memory there. It
	// lives in judgelimits because cmd/compare has to agree with it.
	maxOutputBytes = judgelimits.MaxOutputBytes
	// outputPreviewBytes is read back for the wrong-answer report. It is above
	// the application layer's own preview so its truncation marker stays honest.
	outputPreviewBytes = 4 << 10
	// outputLimitBlocks is what ulimit -f takes: 512-byte blocks, and one more
	// than maxOutputBytes needs so that going over is visible in the file size.
	outputLimitBlocks = maxOutputBytes/512 + 1
	compileTimeout    = 30 * time.Second
	// runGrace is what the worker waits past the in-container timeout before
	// assuming the daemon is stuck and the container has to go. It has to
	// outlast the SIGKILL that timeout(1) sends a second after its deadline,
	// and also cover the two Docker round trips the same deadline wraps.
	runGrace = 5 * time.Second
)

type Session struct {
	container  *pool.Container // nil after Discard by safety net
	pool       *pool.Pool
	docker     dockerExecClient
	langCfg    LanguageExecConfig
	judgingDir string
}

func (s *Session) Compile(ctx context.Context, req appjudge.CompileRequest) (appjudge.CompileResult, error) {
	ctx30, cancel := context.WithTimeout(ctx, compileTimeout)
	defer cancel()

	if _, err := s.docker.CopyToContainer(ctx30, s.container.ID(), client.CopyToContainerOptions{
		DestinationPath: "/sandbox",
		Content:         buildTar("solution."+s.langCfg.Extension, req.SourceCode, modeSource),
	}); err != nil {
		slog.ErrorContext(ctx, "executor: copy source failed", "container_id", s.container.ID(), "error", err)
		return appjudge.CompileResult{}, apperror.NewInternal()
	}

	if s.langCfg.CompileCmd == "" {
		return appjudge.CompileResult{Success: true}, nil
	}

	execRes, err := s.docker.ExecCreate(ctx30, s.container.ID(), client.ExecCreateOptions{
		Cmd:          strings.Fields(s.langCfg.CompileCmd),
		AttachStdout: true,
		AttachStderr: true,
	})
	if err != nil {
		slog.ErrorContext(ctx, "executor: exec create failed", "container_id", s.container.ID(), "error", err)
		return appjudge.CompileResult{}, apperror.NewInternal()
	}

	att, err := s.docker.ExecAttach(ctx30, execRes.ID, client.ExecAttachOptions{})
	if err != nil {
		slog.ErrorContext(ctx, "executor: exec attach failed", "container_id", s.container.ID(), "error", err)
		return appjudge.CompileResult{}, apperror.NewInternal()
	}
	defer att.Conn.Close()

	var outBuf, errBuf bytes.Buffer
	_, _ = stdcopy.StdCopy(&outBuf, &errBuf, att.Reader)

	inspectRes, err := s.docker.ExecInspect(ctx30, execRes.ID, client.ExecInspectOptions{})
	if err != nil {
		slog.ErrorContext(ctx, "executor: exec inspect failed", "container_id", s.container.ID(), "error", err)
		return appjudge.CompileResult{}, apperror.NewInternal()
	}

	return appjudge.CompileResult{
		Success: inspectRes.ExitCode == 0,
		Log:     strutil.Truncate(outBuf.String()+errBuf.String(), maxCompileLogBytes),
	}, nil
}

func (s *Session) RunTestCase(ctx context.Context, req appjudge.RunRequest) (appjudge.RunResult, error) {
	// Straight to the shared volume: the sandbox reads it from there, and so
	// does the checker, instead of each getting its own copy through the API.
	if err := os.WriteFile(judgingInputPath(s.judgingDir), req.Input, judgingFileMode); err != nil {
		slog.ErrorContext(ctx, "executor: write input failed", "error", err)
		return appjudge.RunResult{}, apperror.NewInternal()
	}

	wallBackstopSecs := max(2, (req.TimeLimitMs*2+999)/1000)
	// The write limit sits one block ABOVE maxOutputBytes on purpose: an output
	// of exactly the limit stays legal, and anything longer lands above it, which
	// is what makes the excess visible in the file size. The exit code cannot be
	// used for it — the three runtimes answer SIGXFSZ with 153, 1 and 0.
	// ulimit -c 0 keeps a core dump of a half-gigabyte process off the disk
	// instead of trusting the daemon's default.
	//
	// time wraps timeout and not the other way round: inside, a TLE would kill
	// it too and no measurement would be written. -q keeps the diagnostic line
	// GNU time prints on a non-zero exit out of the file, which is every TLE and
	// every MLE.
	cmd := fmt.Sprintf(
		"ulimit -c 0; ulimit -f %d; /usr/bin/time -q -f %%M -o %s timeout --kill-after=1s %ds %s < %s > %s 2>/dev/null",
		outputLimitBlocks, judgingMemPath(s.judgingDir), wallBackstopSecs, s.langCfg.RunCmd,
		judgingInputPath(s.judgingDir), judgingOutputPath(s.judgingDir),
	)

	safetyCtx, cancel := context.WithTimeout(ctx, time.Duration(wallBackstopSecs)*time.Second+runGrace)
	defer cancel()

	cpuBeforeNs := s.readCPUNanos(ctx)
	execRes, err := s.docker.ExecCreate(safetyCtx, s.container.ID(), client.ExecCreateOptions{
		Cmd: []string{"sh", "-c", cmd},
	})
	if err != nil {
		slog.ErrorContext(ctx, "executor: exec create failed", "container_id", s.container.ID(), "error", err)
		return appjudge.RunResult{}, apperror.NewInternal()
	}

	att, err := s.docker.ExecAttach(safetyCtx, execRes.ID, client.ExecAttachOptions{})
	if err != nil {
		slog.ErrorContext(ctx, "executor: exec attach failed", "container_id", s.container.ID(), "error", err)
		return appjudge.RunResult{}, apperror.NewInternal()
	}

	stop := context.AfterFunc(safetyCtx, func() { att.Conn.Close() })
	defer stop()

	_, _ = io.Copy(io.Discard, att.Reader)

	if safetyCtx.Err() != nil {
		slog.ErrorContext(ctx, "executor: safety net activated", "container_id", s.container.ID())
		s.pool.Discard(ctx, s.container)
		s.container = nil
		return appjudge.RunResult{}, apperror.NewInternal()
	}
	att.Conn.Close()

	inspectRes, err := s.docker.ExecInspect(ctx, execRes.ID, client.ExecInspectOptions{})
	if err != nil {
		slog.ErrorContext(ctx, "executor: exec inspect failed", "container_id", s.container.ID(), "error", err)
		return appjudge.RunResult{}, apperror.NewInternal()
	}

	cpuAfterNs := s.readCPUNanos(ctx)
	cpuTimeMs := 0
	if cpuAfterNs > cpuBeforeNs {
		cpuTimeMs = int((cpuAfterNs - cpuBeforeNs) / 1_000_000)
	}

	preview, exceeded := s.readOutput(ctx)
	result := appjudge.RunResult{
		ExitCode:            inspectRes.ExitCode,
		TimeMs:              cpuTimeMs,
		MemoryKb:            s.readMemoryKb(ctx),
		OutputPreview:       preview,
		OutputLimitExceeded: exceeded,
	}
	// No per-case cleanup: the shell redirection truncates output.txt before the
	// program starts, the worker overwrites input.txt, and Close removes both.
	return result, nil
}

func (s *Session) Close(ctx context.Context) error {
	// Ahead of the container guard on purpose: the safety net nils the container
	// out, and the judging directory would then stay in the volume forever.
	if err := os.RemoveAll(s.judgingDir); err != nil {
		slog.ErrorContext(ctx, "executor: judging directory cleanup failed", "error", err)
	}
	if s.container == nil {
		return nil
	}
	if err := runAndWait(ctx, s.docker, s.container.ID(), []string{"sh", "-c", "rm -rf /sandbox/*"}); err != nil {
		slog.ErrorContext(ctx, "executor: sandbox cleanup failed", "container_id", s.container.ID(), "error", err)
	}
	s.pool.Release(s.container)
	return nil
}

// readCPUNanos returns the container's cumulative CPU time. Memory does not
// come from here: MemoryStats.MaxUsage is a cgroup v1 field, absent on v2, and
// the container is reused across test cases anyway.
func (s *Session) readCPUNanos(ctx context.Context) uint64 {
	statsRes, err := s.docker.ContainerStats(ctx, s.container.ID(), client.ContainerStatsOptions{Stream: false})
	if err != nil {
		return 0
	}
	defer statsRes.Body.Close()
	var stats container.StatsResponse
	if err := json.NewDecoder(statsRes.Body).Decode(&stats); err != nil {
		return 0
	}
	return stats.CPUStats.CPUUsage.TotalUsage
}

// readMemoryKb picks up the peak RSS /usr/bin/time left behind: the run's own,
// isolated from whatever else passed through this container. nil rather than
// zero when there is nothing to read, so no verdict claims a solution used no
// memory at all.
func (s *Session) readMemoryKb(ctx context.Context) *int {
	raw, err := os.ReadFile(judgingMemPath(s.judgingDir))
	if err != nil {
		slog.ErrorContext(ctx, "executor: reading the memory measurement failed", "error", err)
		return nil
	}
	kb, err := strconv.Atoi(strings.TrimSpace(string(raw)))
	if err != nil {
		slog.ErrorContext(ctx, "executor: the memory measurement is not a number", "content", strutil.Truncate(string(raw), 200))
		return nil
	}
	return &kb
}

// readOutput brings back only what the wrong-answer report needs, and whether
// the run wrote past the limit. The output itself stays in the volume for the
// checker to read.
//
// The size is what decides: the write limit sits one block above
// maxOutputBytes, so an output that went over lands above it, whatever the
// runtime did with the signal that stopped it.
func (s *Session) readOutput(ctx context.Context) ([]byte, bool) {
	outputPath := judgingOutputPath(s.judgingDir)
	info, err := os.Stat(outputPath)
	if err != nil {
		slog.ErrorContext(ctx, "executor: output file missing after the run", "error", err)
		return nil, false
	}
	exceeded := info.Size() > maxOutputBytes

	f, err := os.Open(outputPath)
	if err != nil {
		slog.ErrorContext(ctx, "executor: opening the output failed", "error", err)
		return nil, exceeded
	}
	defer f.Close()

	preview := make([]byte, outputPreviewBytes)
	n, err := io.ReadFull(f, preview)
	if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
		slog.ErrorContext(ctx, "executor: reading the output failed", "error", err)
		return nil, exceeded
	}
	return preview[:n], exceeded
}
