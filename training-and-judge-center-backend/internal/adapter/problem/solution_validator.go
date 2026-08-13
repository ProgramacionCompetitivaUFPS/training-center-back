package problem

import (
	"context"

	appJudge "github.com/training-judge-center/backend/internal/application/judge"
	appProblem "github.com/training-judge-center/backend/internal/application/problem"
)

// SolutionValidator is the only file in the system that imports both
// application/judge and application/problem — its entire job is translating
// between the two, so neither application package has to import the other.
type SolutionValidator struct {
	uc *appJudge.ValidateSolutionsUseCase
}

func NewSolutionValidator(uc *appJudge.ValidateSolutionsUseCase) *SolutionValidator {
	return &SolutionValidator{uc: uc}
}

func (v *SolutionValidator) Validate(ctx context.Context, problemID string) (*appProblem.SolutionValidationResult, error) {
	out, err := v.uc.Execute(ctx, appJudge.ValidateSolutionsInput{ProblemID: problemID})
	if err != nil {
		return nil, err
	}

	result := &appProblem.SolutionValidationResult{
		SampleCases:     out.SampleCases,
		SecretCases:     out.SecretCases,
		SolutionsTested: out.SolutionsTested,
		Passed:          out.Passed,
	}
	if out.Failure != nil {
		result.Failure = &appProblem.SolutionValidationFailure{
			FileKey:     out.Failure.FileKey,
			Kind:        appProblem.SolutionFailureKind(out.Failure.Kind),
			CompileLog:  out.Failure.CompileLog,
			TestCase:    out.Failure.TestCase,
			TimeMs:      out.Failure.TimeMs,
			TimeLimitMs: out.Failure.TimeLimitMs,
			Expected:    out.Failure.Expected,
			Actual:      out.Failure.Actual,
		}
	}
	return result, nil
}
