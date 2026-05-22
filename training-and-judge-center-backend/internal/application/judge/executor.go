package judge

import (
	"context"

	"github.com/training-judge-center/backend/internal/domain/submission"
)

type CompileRequest struct {
	Language   submission.Language
	SourceCode []byte
}

type CompileResult struct {
	Success bool
	Log     string
}

type RunRequest struct {
	Input       []byte
	TimeLimitMs int
	MemoryKb    int
}

type RunResult struct {
	ExitCode int
	TimeMs   int
	MemoryKb int
	Output   []byte
}

type ExecutionSession interface {
	Compile(ctx context.Context, req CompileRequest) (CompileResult, error)
	RunTestCase(ctx context.Context, req RunRequest) (RunResult, error)
	Close(ctx context.Context) error
}

type Executor interface {
	BeginSession(ctx context.Context, language submission.Language) (ExecutionSession, error)
}
