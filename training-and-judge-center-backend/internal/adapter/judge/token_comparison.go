package judge

import (
	"context"
	"log/slog"
	"os"
	"strings"

	appjudge "github.com/training-judge-center/backend/internal/application/judge"
	"github.com/training-judge-center/backend/pkg/apperror"
)

var _ appjudge.CheckerSession = (*tokenCheckerSession)(nil)

// tokenCheckerSession is the no-custom-checker path, still running in the worker.
// Step 3 of the shared volume work replaces it with a light pool container
// running the compare image (D6), and this file goes away with it.
type tokenCheckerSession struct {
	judgingDir string
}

// Reading the whole output back into the worker is exactly what the shared
// volume exists to avoid. It survives only until the compare image takes over,
// which is the very next step.
func (s *tokenCheckerSession) Check(ctx context.Context, expectedOutput []byte) (appjudge.CheckResult, error) {
	contestantOutput, err := os.ReadFile(judgingOutputPath(s.judgingDir))
	if err != nil {
		slog.ErrorContext(ctx, "token_comparison: reading the contestant's output failed", "error", err)
		return appjudge.CheckResult{}, apperror.NewInternal()
	}
	return tokenCompare(expectedOutput, contestantOutput), nil
}

func (s *tokenCheckerSession) Close(context.Context) error { return nil }

// tokenCompare splits both outputs into whitespace-delimited tokens and
// compares them element by element. strings.Fields handles \r\n, multiple
// spaces, and trailing whitespace, so semantically equivalent outputs
// produced on different platforms are treated as equal.
func tokenCompare(expected, contestant []byte) appjudge.CheckResult {
	expTokens := strings.Fields(string(expected))
	conTokens := strings.Fields(string(contestant))
	if len(expTokens) != len(conTokens) {
		return appjudge.CheckResult{Accepted: false}
	}
	for i := range expTokens {
		if expTokens[i] != conTokens[i] {
			return appjudge.CheckResult{Accepted: false}
		}
	}
	return appjudge.CheckResult{Accepted: true}
}
