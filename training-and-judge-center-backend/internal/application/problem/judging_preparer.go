package problem

import "context"

type JudgingFailureKind string

const (
	JudgingFailureCheckerCompileError   JudgingFailureKind = "CHECKER_COMPILE_ERROR"
	JudgingFailureValidatorCompileError JudgingFailureKind = "VALIDATOR_COMPILE_ERROR"
	JudgingFailureInputRejected         JudgingFailureKind = "INPUT_REJECTED"
)

// JudgingPreparationFailure mirrors judge.PrepareJudgingFailure — kept as a
// separate local type so this package never imports application/judge.
type JudgingPreparationFailure struct {
	Kind     JudgingFailureKind
	FileKey  string
	Log      string
	TestCase string
	Reason   string
}

type JudgingPreparationResult struct {
	CheckerCompiledKey   string
	ValidatorCompiledKey string
	Failure              *JudgingPreparationFailure
}

// JudgingPreparer compiles a problem's checker/validator (if uploaded) and,
// if the validator compiled, runs it against every test case's input.
type JudgingPreparer interface {
	Prepare(ctx context.Context, problemID, slug string, candidate *JudgingCandidate) (*JudgingPreparationResult, error)
}

// JudgingCandidate mirrors judge.JudgingCandidate; empty fields keep what the problem has.
type JudgingCandidate struct {
	TestCasesKey       string
	CheckerFileKey     string
	CheckerLanguage    string
	CheckerCompiledKey string
}
