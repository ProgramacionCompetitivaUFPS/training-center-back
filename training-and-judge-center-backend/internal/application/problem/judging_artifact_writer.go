package problem

import (
	"context"
	"time"
)

type JudgingArtifactWriter interface {
	SetCheckerCompiledKey(ctx context.Context, problemID, compiledKey string, now time.Time) error
	SetValidatorCompiledKey(ctx context.Context, problemID, compiledKey string, now time.Time) error
}
