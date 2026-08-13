package judge

import (
	"bytes"
	"context"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"

	appJudge "github.com/training-judge-center/backend/internal/application/judge"
	"github.com/training-judge-center/backend/pkg/apperror"
)

func compileCpp(ctx context.Context, dir string, req appJudge.CompileArtifactRequest) (appJudge.CompileArtifactResult, error) {
	sourcePath := filepath.Join(dir, filepath.Base(req.Filename))
	binaryPath := filepath.Join(dir, "artifact")

	if err := os.WriteFile(sourcePath, req.SourceCode, 0o644); err != nil {
		slog.ErrorContext(ctx, "native_compiler: failed to write cpp source", "error", err)
		return appJudge.CompileArtifactResult{}, apperror.NewInternal()
	}

	var stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, "g++", "-std=c++20", "-O2", "-o", binaryPath, sourcePath)
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if isTimeoutErr(ctx, err) {
			slog.ErrorContext(ctx, "native_compiler: g++ timed out", "error", ctx.Err())
			return appJudge.CompileArtifactResult{}, apperror.NewInternal()
		}
		if _, ok := err.(*exec.ExitError); ok {
			return appJudge.CompileArtifactResult{Success: false, Log: stderr.String()}, nil
		}
		slog.ErrorContext(ctx, "native_compiler: failed to run g++", "error", err)
		return appJudge.CompileArtifactResult{}, apperror.NewInternal()
	}

	artifact, err := os.ReadFile(binaryPath)
	if err != nil {
		slog.ErrorContext(ctx, "native_compiler: failed to read compiled cpp artifact", "error", err)
		return appJudge.CompileArtifactResult{}, apperror.NewInternal()
	}
	return appJudge.CompileArtifactResult{Success: true, Artifact: artifact}, nil
}
