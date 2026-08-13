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

func compilePython(ctx context.Context, dir string, req appJudge.CompileArtifactRequest) (appJudge.CompileArtifactResult, error) {
	sourcePath := filepath.Join(dir, filepath.Base(req.Filename))

	if err := os.WriteFile(sourcePath, req.SourceCode, 0o644); err != nil {
		slog.ErrorContext(ctx, "native_compiler: failed to write python source", "error", err)
		return appJudge.CompileArtifactResult{}, apperror.NewInternal()
	}

	var stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, "python3", "-m", "py_compile", sourcePath)
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if isTimeoutErr(ctx, err) {
			slog.ErrorContext(ctx, "native_compiler: python3 timed out", "error", ctx.Err())
			return appJudge.CompileArtifactResult{}, apperror.NewInternal()
		}
		if _, ok := err.(*exec.ExitError); ok {
			return appJudge.CompileArtifactResult{Success: false, Log: stderr.String()}, nil
		}
		slog.ErrorContext(ctx, "native_compiler: failed to run python3", "error", err)
		return appJudge.CompileArtifactResult{}, apperror.NewInternal()
	}

	// py_compile's .pyc is version/interpreter-specific and not worth
	// keeping — the artifact is the source itself, now known to at least
	// parse; it's what gets run later (python3 script.py).
	return appJudge.CompileArtifactResult{Success: true, Artifact: req.SourceCode}, nil
}
