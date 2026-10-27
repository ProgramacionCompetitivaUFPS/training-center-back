package judge

import (
	"context"

	"github.com/training-judge-center/backend/internal/domain/submission"
)

type ValidatorRunResult struct {
	Accepted bool
	Message  string // the validator's stderr, set when rejected
}

// ValidatorRunner opens a session that runs one compiled validator against many
// inputs. Opening claims a sandbox container and injects the artifact once, so
// that cost is paid per problem instead of per test case.
type ValidatorRunner interface {
	BeginValidating(ctx context.Context, validatorPath string, language submission.Language) (ValidatorSession, error)
}

// ValidatorSession follows the testlib convention: the input arrives on stdin
// and exit code 0 means accepted.
type ValidatorSession interface {
	Validate(ctx context.Context, input []byte) (ValidatorRunResult, error)
	Close(ctx context.Context) error
}
