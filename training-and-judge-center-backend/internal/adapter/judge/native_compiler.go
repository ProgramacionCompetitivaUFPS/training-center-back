package judge

import (
	"context"
	"log/slog"
	"os"
	"time"

	appJudge "github.com/training-judge-center/backend/internal/application/judge"
	"github.com/training-judge-center/backend/pkg/apperror"
)

var _ appJudge.NativeCompiler = (*NativeCompiler)(nil)

type compileFunc func(ctx context.Context, dir string, req appJudge.CompileArtifactRequest) (appJudge.CompileArtifactResult, error)

var nativeCompilers = map[string]compileFunc{
	"cpp20":     compileCpp,
	"java17":    compileJava,
	"python310": compilePython,
}

type NativeCompiler struct {
	timeout time.Duration
}

func NewNativeCompiler() *NativeCompiler {
	return &NativeCompiler{timeout: trustedSubprocessTimeout}
}

func (c *NativeCompiler) Compile(ctx context.Context, req appJudge.CompileArtifactRequest) (appJudge.CompileArtifactResult, error) {
	fn, ok := nativeCompilers[req.Language.String()]
	if !ok {
		slog.ErrorContext(ctx, "native_compiler: unsupported language", "language", req.Language.String())
		return appJudge.CompileArtifactResult{}, apperror.NewInternal()
	}

	tmpDir, err := os.MkdirTemp("", "native-compile-*")
	if err != nil {
		slog.ErrorContext(ctx, "native_compiler: failed to create temp dir", "error", err)
		return appJudge.CompileArtifactResult{}, apperror.NewInternal()
	}
	defer os.RemoveAll(tmpDir)

	compileCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	return fn(compileCtx, tmpDir, req)
}
