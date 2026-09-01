package problem

import (
	"context"

	appJudge "github.com/training-judge-center/backend/internal/application/judge"
	appProblem "github.com/training-judge-center/backend/internal/application/problem"
)

// JudgingPreparer is another translator like SolutionValidator — it's the
// only file that imports both application/judge and application/problem.
type JudgingPreparer struct {
	uc *appJudge.PrepareJudgingUseCase
}

func NewJudgingPreparer(uc *appJudge.PrepareJudgingUseCase) *JudgingPreparer {
	return &JudgingPreparer{uc: uc}
}

func (p *JudgingPreparer) Prepare(ctx context.Context, problemID, slug string) (*appProblem.JudgingPreparationResult, error) {
	out, err := p.uc.Execute(ctx, appJudge.PrepareJudgingInput{ProblemID: problemID, Slug: slug})
	if err != nil {
		return nil, err
	}

	result := &appProblem.JudgingPreparationResult{
		CheckerCompiledKey:   out.CheckerCompiledKey,
		ValidatorCompiledKey: out.ValidatorCompiledKey,
	}
	if out.Failure != nil {
		result.Failure = &appProblem.JudgingPreparationFailure{
			Kind:     appProblem.JudgingFailureKind(out.Failure.Kind),
			FileKey:  out.Failure.FileKey,
			Log:      out.Failure.Log,
			TestCase: out.Failure.TestCase,
			Reason:   out.Failure.Reason,
		}
	}
	return result, nil
}
