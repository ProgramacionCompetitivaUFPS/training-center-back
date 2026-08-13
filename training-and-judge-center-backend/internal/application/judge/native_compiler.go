package judge

import (
	"context"

	"github.com/training-judge-center/backend/internal/domain/submission"
)

type CompileArtifactRequest struct {
	// Filename is the original uploaded name, e.g. "Checker.java" — javac
	// requires it to match the public class name, so writing the source to
	// disk under this same name is what makes that work without any extra
	// bookkeeping.
	Filename   string
	Language   submission.Language
	SourceCode []byte
}

type CompileArtifactResult struct {
	Success  bool
	Log      string
	Artifact []byte // nil if !Success
}

// NativeCompiler compiles checker/validator source directly on the worker's
// own filesystem, no sandbox — that code is trusted (the problem setter's),
// unlike a contestant's submission.
type NativeCompiler interface {
	Compile(ctx context.Context, req CompileArtifactRequest) (CompileArtifactResult, error)
}
