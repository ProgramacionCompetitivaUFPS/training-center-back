package judge

import (
	"context"
	"strings"

	appjudge "github.com/training-judge-center/backend/internal/application/judge"
)

var _ appjudge.CheckerSession = (*tokenCheckerSession)(nil)

// tokenCheckerSession is the no-custom-checker path, still running in the worker.
// Step 7 replaces it with a light pool container running the compare image (D6),
// and this file goes away with it.
type tokenCheckerSession struct{}

func (s *tokenCheckerSession) Check(_ context.Context, req appjudge.CheckRequest) (appjudge.CheckResult, error) {
	return tokenCompare(req.ExpectedOutput, req.ContestantOutput), nil
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
