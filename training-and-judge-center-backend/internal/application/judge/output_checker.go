package judge

import (
	"context"

	"github.com/training-judge-center/backend/internal/domain/submission"
)

type CheckRequest struct {
	Input            []byte
	ExpectedOutput   []byte
	ContestantOutput []byte
}

type CheckResult struct {
	Accepted bool
	Message  string
}

// OutputChecker opens a session that checks every output of one judging with the
// same checker, claiming its container and injecting the artifact once.
type OutputChecker interface {
	// An empty checkerPath means no custom checker: token comparison, no container.
	BeginChecking(ctx context.Context, checkerPath string, language submission.Language) (CheckerSession, error)
}

// CheckerSession checks one test case per call. A rejected output is a verdict,
// not an error.
type CheckerSession interface {
	Check(ctx context.Context, req CheckRequest) (CheckResult, error)
	Close(ctx context.Context) error
}
