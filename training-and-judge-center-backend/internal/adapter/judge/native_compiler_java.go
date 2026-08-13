package judge

import (
	"bytes"
	"context"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	appJudge "github.com/training-judge-center/backend/internal/application/judge"
	"github.com/training-judge-center/backend/pkg/apperror"
)

// compileJava relies on Java's own rule that a source file's name must match
// its public class name — so writing the source to disk under its original
// filename is what lets us find the resulting .class afterward, without
// tracking a class name anywhere.
func compileJava(ctx context.Context, dir string, req appJudge.CompileArtifactRequest) (appJudge.CompileArtifactResult, error) {
	filename := filepath.Base(req.Filename)
	sourcePath := filepath.Join(dir, filename)

	if err := os.WriteFile(sourcePath, req.SourceCode, 0o644); err != nil {
		slog.ErrorContext(ctx, "native_compiler: failed to write java source", "error", err)
		return appJudge.CompileArtifactResult{}, apperror.NewInternal()
	}

	var stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, "javac", "-encoding", "UTF-8", "-d", dir, sourcePath)
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if isTimeoutErr(ctx, err) {
			slog.ErrorContext(ctx, "native_compiler: javac timed out", "error", ctx.Err())
			return appJudge.CompileArtifactResult{}, apperror.NewInternal()
		}
		if _, ok := err.(*exec.ExitError); ok {
			return appJudge.CompileArtifactResult{Success: false, Log: stderr.String()}, nil
		}
		slog.ErrorContext(ctx, "native_compiler: failed to run javac", "error", err)
		return appJudge.CompileArtifactResult{}, apperror.NewInternal()
	}

	className := strings.TrimSuffix(filename, filepath.Ext(filename))
	classPath := filepath.Join(dir, className+".class")
	artifact, err := os.ReadFile(classPath)
	if err != nil {
		slog.ErrorContext(ctx, "native_compiler: failed to read compiled java artifact", "error", err, "class_path", classPath)
		return appJudge.CompileArtifactResult{}, apperror.NewInternal()
	}
	return appJudge.CompileArtifactResult{Success: true, Artifact: artifact}, nil
}
