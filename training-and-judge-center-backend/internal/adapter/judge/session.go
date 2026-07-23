package judge

import (
	"archive/tar"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"time"

	"github.com/moby/moby/api/pkg/stdcopy"
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/client"

	"github.com/training-judge-center/backend/internal/adapter/judge/pool"
	appjudge "github.com/training-judge-center/backend/internal/application/judge"
	"github.com/training-judge-center/backend/pkg/apperror"
)

const (
	maxCompileLogBytes = 10 * 1024
	maxOutputBytes     = 64 << 20
	compileTimeout     = 30 * time.Second
)

type Session struct {
	container *pool.Container // nil after Discard by safety net
	pool      *pool.Pool
	docker    dockerExecClient
	langCfg   LanguageExecConfig
}

func (s *Session) Compile(ctx context.Context, req appjudge.CompileRequest) (appjudge.CompileResult, error) {
	ctx30, cancel := context.WithTimeout(ctx, compileTimeout)
	defer cancel()

	if _, err := s.docker.CopyToContainer(ctx30, s.container.ID(), client.CopyToContainerOptions{
		DestinationPath: "/sandbox",
		Content:         buildTar("solution."+s.langCfg.Extension, req.SourceCode),
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
		Log:     truncateString(outBuf.String()+errBuf.String(), maxCompileLogBytes),
	}, nil
}

func (s *Session) RunTestCase(ctx context.Context, req appjudge.RunRequest) (appjudge.RunResult, error) {
	if _, err := s.docker.CopyToContainer(ctx, s.container.ID(), client.CopyToContainerOptions{
		DestinationPath: "/sandbox",
		Content:         buildTar("input.txt", req.Input),
	}); err != nil {
		slog.ErrorContext(ctx, "executor: copy input failed", "container_id", s.container.ID(), "error", err)
		return appjudge.RunResult{}, apperror.NewInternal()
	}

	wallBackstopSecs := max(2, (req.TimeLimitMs*2+999)/1000)
	cmd := fmt.Sprintf(
		"timeout --kill-after=1s %ds %s < /sandbox/input.txt > /sandbox/output.txt 2>/dev/null",
		wallBackstopSecs, s.langCfg.RunCmd,
	)

	safetyCtx, cancel := context.WithTimeout(ctx, time.Duration(wallBackstopSecs+2)*time.Second)
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
		ExitCode: inspectRes.ExitCode,
		TimeMs:   cpuTimeMs,
		MemoryKb: memoryKb,
		Output:   s.copyOutput(ctx),
	}
	s.cleanup(ctx)
	return result, nil
}

func (s *Session) Close(ctx context.Context) error {
	if s.container == nil {
		return nil
	}
	s.runWait(ctx, []string{"sh", "-c", "rm -rf /sandbox/*"})
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

func (s *Session) copyOutput(ctx context.Context) []byte {
	res, err := s.docker.CopyFromContainer(ctx, s.container.ID(), client.CopyFromContainerOptions{
		SourcePath: "/sandbox/output.txt",
	})
	if err != nil {
		return nil
	}
	defer res.Content.Close()
	return extractFirstFile(res.Content, maxOutputBytes)
}

func (s *Session) cleanup(ctx context.Context) {
	s.runWait(ctx, []string{"sh", "-c", "rm -f /sandbox/input.txt /sandbox/output.txt"})
}

func (s *Session) runWait(ctx context.Context, cmd []string) {
	execRes, err := s.docker.ExecCreate(ctx, s.container.ID(), client.ExecCreateOptions{Cmd: cmd})
	if err != nil {
		return
	}
	att, err := s.docker.ExecAttach(ctx, execRes.ID, client.ExecAttachOptions{})
	if err != nil {
		return
	}
	_, _ = io.Copy(io.Discard, att.Reader)
	att.Conn.Close()
}

func buildTar(filename string, content []byte) io.Reader {
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	_ = tw.WriteHeader(&tar.Header{Name: filename, Mode: 0644, Size: int64(len(content))})
	_, _ = tw.Write(content)
	_ = tw.Close()
	return &buf
}

func extractFirstFile(r io.Reader, maxBytes int64) []byte {
	tr := tar.NewReader(r)
	if _, err := tr.Next(); err != nil {
		return nil
	}
	data, _ := io.ReadAll(&io.LimitedReader{R: tr, N: maxBytes})
	return data
}

func truncateString(s string, maxBytes int) string {
	if len(s) <= maxBytes {
		return s
	}
	return s[:maxBytes]
}
