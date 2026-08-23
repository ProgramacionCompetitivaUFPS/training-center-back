package judge

import (
	"context"

	"github.com/training-judge-center/backend/internal/domain/submission"
)

type CheckResult struct {
	Accepted bool
	Message  string
}

// OutputChecker opens a session that checks every output of one judging with the
// same checker, claiming its container and injecting the artifact once.
type OutputChecker interface {
	// An empty checkerPath means no custom checker: token comparison, no container.
	// judgingID must be the one the execution session was opened with: that is
	// how both sandboxes end up looking at the same files.
	BeginChecking(ctx context.Context, checkerPath string, language submission.Language, judgingID string) (CheckerSession, error)
}

// CheckerSession checks one test case per call. A rejected output is a verdict,
// not an error.
type CheckerSession interface {
	// Check reads whatever the paired session's last RunTestCase left behind.
	// The jury's answer travels as bytes because it is the one file the
	// contestant's container must never be able to reach.
	Check(ctx context.Context, expectedOutput []byte) (CheckResult, error)
	Close(ctx context.Context) error
}
