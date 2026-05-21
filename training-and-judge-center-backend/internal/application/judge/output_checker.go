package judge

import "context"

type CheckRequest struct {
	Input            []byte
	ExpectedOutput   []byte
	ContestantOutput []byte
	CheckerPath      string
}

type CheckResult struct {
	Accepted bool
	Message  string
}

type OutputChecker interface {
	Check(ctx context.Context, req CheckRequest) (CheckResult, error)
}
