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
	// CheckerLanguage is only meaningful when CheckerPath is set: it picks which
	// light pool image runs the artifact.
	CheckerLanguage submission.Language
}

type ProblemProvider interface {
	GetLimits(ctx context.Context, problemID string) (ProblemLimits, error)
}
