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
}

type RunResult struct {
	ExitCode int
	TimeMs   int
	// MemoryKb is nil when the run produced no measurement. Reporting a zero
	// instead would say the solution used no memory at all.
	MemoryKb *int
	// OutputPreview is the first few KB of what the contestant printed, enough
	// for the wrong-answer report. The output itself never leaves the sandbox.
	OutputPreview []byte
}

type ExecutionSession interface {
	Compile(ctx context.Context, req CompileRequest) (CompileResult, error)
	RunTestCase(ctx context.Context, req RunRequest) (RunResult, error)
	Close(ctx context.Context) error
}

type Executor interface {
	// memoryKb is the problem's limit. It belongs here and not in RunRequest
	// because it is constant for the whole judging and applying it costs a
	// container reconfiguration.
	//
	// judgingID names the directory this judging's files live in. Opaque here,
	// and unguessable by contract: the name is what isolates one judging.
	BeginSession(ctx context.Context, language submission.Language, memoryKb int, judgingID string) (ExecutionSession, error)
}
