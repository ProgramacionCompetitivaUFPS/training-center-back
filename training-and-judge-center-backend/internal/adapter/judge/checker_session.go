package judge

import (
	"context"
	"fmt"
	"log/slog"

	appjudge "github.com/training-judge-center/backend/internal/application/judge"
	"github.com/training-judge-center/backend/pkg/apperror"
)

var _ appjudge.CheckerSession = (*CheckerSession)(nil)

// checkerAnswerPath is the only file of the three that still travels through the
// API: D7 keeps the jury's answer out of the shared volume, where the
// contestant's own container could read it and print it.
const checkerAnswerPath = "/sandbox/answer.txt"

type CheckerSession struct {
	artifactSession
	judgingDir string
}

// Check runs the checker over the input and the output the paired session left
// in the judging directory. A non-zero exit is the checker rejecting the
// output, which is a verdict, not a failure of ours.
func (s *CheckerSession) Check(ctx context.Context, expectedOutput []byte) (appjudge.CheckResult, error) {
	if s.container == nil {
		slog.ErrorContext(ctx, "checker_session: check called on a discarded session")
		return appjudge.CheckResult{}, apperror.NewInternal()
	}

	if err := s.writeFile(ctx, checkerAnswerPath, expectedOutput); err != nil {
		return appjudge.CheckResult{}, err
	}

	// testlib's argument order: input, the contestant's output, the jury's answer.
	exitCode, stderr, err := s.run(ctx, fmt.Sprintf("%s %s %s %s",
		s.runCmd, judgingInputPath(s.judgingDir), judgingOutputPath(s.judgingDir), checkerAnswerPath))
	if err != nil {
		return appjudge.CheckResult{}, err
	}
	if exitCode != 0 {
		return appjudge.CheckResult{Accepted: false, Message: stderr}, nil
	}
	return appjudge.CheckResult{Accepted: true}, nil
}
