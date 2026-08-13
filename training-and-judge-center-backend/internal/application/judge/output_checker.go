package judge

import (
	"context"

	"github.com/training-judge-center/backend/internal/domain/submission"
)

type CheckRequest struct {
	Input            []byte
	ExpectedOutput   []byte
	ContestantOutput []byte
	CheckerPath      string
	// CheckerLanguage/CheckerFilename are only meaningful when CheckerPath
	// is set — see ProblemLimits for why CheckerFilename is needed.
	CheckerLanguage submission.Language
	CheckerFilename string
}

type CheckResult struct {
	Accepted bool
	Message  string
}

type OutputChecker interface {
	Check(ctx context.Context, req CheckRequest) (CheckResult, error)
}
