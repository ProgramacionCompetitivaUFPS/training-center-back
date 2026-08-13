package problem

import "context"

type SolutionFailureKind string

const (
	SolutionFailureCompileError        SolutionFailureKind = "COMPILE_ERROR"
	SolutionFailureWrongAnswer         SolutionFailureKind = "WRONG_ANSWER"
	SolutionFailureTimeLimitExceeded   SolutionFailureKind = "TIME_LIMIT_EXCEEDED"
	SolutionFailureMemoryLimitExceeded SolutionFailureKind = "MEMORY_LIMIT_EXCEEDED"
	SolutionFailureRuntimeError        SolutionFailureKind = "RUNTIME_ERROR"
)

// SolutionValidationFailure mirrors judge.SolutionFailure — kept as a
// separate local type so this package never imports application/judge.
type SolutionValidationFailure struct {
	FileKey     string
	Kind        SolutionFailureKind
	CompileLog  string
	TestCase    string
	TimeMs      int
	TimeLimitMs int
	Expected    []byte
	Actual      []byte
}

type SolutionValidationResult struct {
	SampleCases     int
	SecretCases     int
	SolutionsTested int
	Passed          bool
	Failure         *SolutionValidationFailure
}

// SolutionValidator is implemented by an adapter that delegates to judge's
// execution engine and translates its result into this package's own types.
type SolutionValidator interface {
	Validate(ctx context.Context, problemID string) (*SolutionValidationResult, error)
}
