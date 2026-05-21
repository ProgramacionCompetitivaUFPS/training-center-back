package judge

import "context"

type ProblemLimits struct {
	TimeLimitMs      int
	MemoryKb         int
	HasCustomChecker bool
	CheckerPath      string
}

type ProblemProvider interface {
	GetLimits(ctx context.Context, problemID string) (ProblemLimits, error)
}
