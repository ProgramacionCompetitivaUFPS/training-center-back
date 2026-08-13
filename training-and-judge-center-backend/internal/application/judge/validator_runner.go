package judge

import (
	"context"

	"github.com/training-judge-center/backend/internal/domain/submission"
)

type ValidatorRunRequest struct {
	// Filename is the original uploaded name — needed to derive the class
	// name for java, same reasoning as CompileArtifactRequest.
	Filename string
	Language submission.Language
	Artifact []byte // the compiled validator, from NativeCompiler
	Input    []byte // the test case input to validate
}

type ValidatorRunResult struct {
	Accepted bool
	Message  string // the validator's stderr, set when rejected
}

// ValidatorRunner runs a compiled validator against one input, testlib
// convention: input on stdin, exit code 0 means accepted.
type ValidatorRunner interface {
	Run(ctx context.Context, req ValidatorRunRequest) (ValidatorRunResult, error)
}
