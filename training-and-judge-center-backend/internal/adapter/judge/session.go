package judge

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/moby/moby/api/pkg/stdcopy"
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/client"

	"github.com/training-judge-center/backend/internal/adapter/judge/pool"
	appjudge "github.com/training-judge-center/backend/internal/application/judge"
	"github.com/training-judge-center/backend/pkg/apperror"
	"github.com/training-judge-center/backend/pkg/strutil"
)

const (
	maxCompileLogBytes = 10 * 1024
	// maxOutputBytes is what a checker has to hold in a light pool container;
	// 64 MiB made every language run out of memory there. Nothing enforces it
	// yet: step 5 of the shared volume work turns it into a verdict.
	maxOutputBytes = 8 << 20
	// outputPreviewBytes is read back for the wrong-answer report. It is above
	// the application layer's own preview so its truncation marker stays honest.
	outputPreviewBytes = 4 << 10
	compileTimeout     = 30 * time.Second
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
	cmd := fmt.Sprintf(
		"timeout --kill-after=1s %ds %s < %s > %s 2>/dev/null",
		wallBackstopSecs, s.langCfg.RunCmd,
		judgingInputPath(s.judgingDir), judgingOutputPath(s.judgingDir),
	)

	safetyCtx, cancel := context.WithTimeout(ctx, time.Duration(wallBackstopSecs)*time.Second+runGrace)
	defer cancel()

	cpuBeforeNs, _ := s.readStats(ctx)
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

	cpuAfterNs, memoryKb := s.readStats(ctx)
	cpuTimeMs := 0
	if cpuAfterNs > cpuBeforeNs {
		cpuTimeMs = int((cpuAfterNs - cpuBeforeNs) / 1_000_000)
	}

	result := appjudge.RunResult{
		ExitCode:      inspectRes.ExitCode,
		TimeMs:        cpuTimeMs,
		MemoryKb:      memoryKb,
		OutputPreview: s.readOutputPreview(ctx),
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

func (s *Session) readStats(ctx context.Context) (uint64, int) {
	statsRes, err := s.docker.ContainerStats(ctx, s.container.ID(), client.ContainerStatsOptions{Stream: false})
	if err != nil {
		return 0, 0
	}
	defer statsRes.Body.Close()
	var stats container.StatsResponse
	if err := json.NewDecoder(statsRes.Body).Decode(&stats); err != nil {
		return 0, 0
	}
	return stats.CPUStats.CPUUsage.TotalUsage, int(stats.MemoryStats.MaxUsage / 1024)
}

// readOutputPreview brings back only what the wrong-answer report needs. The
// output itself stays in the volume for the checker to read.
func (s *Session) readOutputPreview(ctx context.Context) []byte {
	outputPath := judgingOutputPath(s.judgingDir)
	info, err := os.Stat(outputPath)
	if err != nil {
		slog.ErrorContext(ctx, "executor: output file missing after the run", "error", err)
		return nil
	}
	// Nothing stops a program from printing past the limit until step 5 makes it
	// a verdict, so until then it at least leaves a trace.
	if info.Size() > maxOutputBytes {
		slog.WarnContext(ctx, "executor: contestant output ran past the limit",
			"limit_bytes", maxOutputBytes, "size_bytes", info.Size())
	}

	f, err := os.Open(outputPath)
	if err != nil {
		slog.ErrorContext(ctx, "executor: opening the output failed", "error", err)
		return nil
	}
	defer f.Close()

	preview := make([]byte, outputPreviewBytes)
	n, err := io.ReadFull(f, preview)
	if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
		slog.ErrorContext(ctx, "executor: reading the output failed", "error", err)
		return nil
	}
	return preview[:n]
}
