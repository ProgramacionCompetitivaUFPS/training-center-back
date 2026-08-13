package judge

import (
	"context"

	"github.com/training-judge-center/backend/internal/domain/submission"
)

type ProblemLimits struct {
	TimeLimitMs      int
	MemoryKb         int
	HasCustomChecker bool
	// CheckerPath points at the compiled checker artifact, not its source —
	// empty (forcing token comparison) until the checker has been compiled,
	// which publish validation guarantees for any PUBLISHED problem.
	CheckerPath string
	// CheckerLanguage/CheckerFilename are only meaningful when CheckerPath
	// is set. CheckerFilename is needed to invoke a compiled Java checker —
	// its class name must match the original uploaded filename.
	CheckerLanguage submission.Language
	CheckerFilename string
}

type ProblemProvider interface {
	GetLimits(ctx context.Context, problemID string) (ProblemLimits, error)
}
