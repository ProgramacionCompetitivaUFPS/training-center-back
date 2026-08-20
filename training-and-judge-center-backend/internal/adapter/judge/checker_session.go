package judge

import (
	"context"
	"fmt"
	"log/slog"

	appjudge "github.com/training-judge-center/backend/internal/application/judge"
	"github.com/training-judge-center/backend/pkg/apperror"
)

var _ appjudge.CheckerSession = (*CheckerSession)(nil)

// The two files only a checker reads, next to the shared input: testlib calls
// them ouf (the contestant's) and ans (the jury's).
const (
	checkerOutputPath = "/sandbox/output.txt"
	checkerAnswerPath = "/sandbox/answer.txt"
)

type CheckerSession struct {
	artifactSession
}

// Check writes the test case's three files into the sandbox and runs the checker
// over them. A non-zero exit is the checker rejecting the output, which is a
// verdict, not a failure of ours.
func (s *CheckerSession) Check(ctx context.Context, req appjudge.CheckRequest) (appjudge.CheckResult, error) {
	if s.container == nil {
		slog.ErrorContext(ctx, "checker_session: check called on a discarded session")
		return appjudge.CheckResult{}, apperror.NewInternal()
	}

	for _, f := range []struct {
		path    string
		content []byte
	}{
		{sandboxInputPath, req.Input},
		{checkerOutputPath, req.ContestantOutput},
		{checkerAnswerPath, req.ExpectedOutput},
	} {
		if err := s.writeFile(ctx, f.path, f.content); err != nil {
			return appjudge.CheckResult{}, err
		}
	}

	// testlib's argument order: input, the contestant's output, the jury's answer.
	exitCode, stderr, err := s.run(ctx, fmt.Sprintf("%s %s %s %s",
		s.runCmd, sandboxInputPath, checkerOutputPath, checkerAnswerPath))
	if err != nil {
		return appjudge.CheckResult{}, err
	}
	if exitCode != 0 {
		return appjudge.CheckResult{Accepted: false, Message: stderr}, nil
	}
	return appjudge.CheckResult{Accepted: true}, nil
}
