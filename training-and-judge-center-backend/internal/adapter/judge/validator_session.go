package judge

import (
	"context"
	"fmt"
	"io"
	"log/slog"

	appjudge "github.com/training-judge-center/backend/internal/application/judge"
	"github.com/training-judge-center/backend/pkg/apperror"
)

var _ appjudge.ValidatorSession = (*ValidatorSession)(nil)

type ValidatorSession struct {
	artifactSession
}

// Validate feeds one input to the validator on stdin. A non-zero exit is the
// validator rejecting the test case, which is a result, not a failure of ours.
func (s *ValidatorSession) Validate(ctx context.Context, input io.Reader, size int64) (appjudge.ValidatorRunResult, error) {
	if s.container == nil {
		slog.ErrorContext(ctx, "validator_session: validate called on a discarded session")
		return appjudge.ValidatorRunResult{}, apperror.NewInternal()
	}

	if err := s.writeStream(ctx, sandboxInputPath, input, size); err != nil {
		return appjudge.ValidatorRunResult{}, err
	}

	exitCode, stderr, err := s.run(ctx, fmt.Sprintf("%s < %s", s.runCmd, sandboxInputPath))
	if err != nil {
		return appjudge.ValidatorRunResult{}, err
	}
	if exitCode != 0 {
		return appjudge.ValidatorRunResult{Accepted: false, Message: stderr}, nil
	}
	return appjudge.ValidatorRunResult{Accepted: true}, nil
}
