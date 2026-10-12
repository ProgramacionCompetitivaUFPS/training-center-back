package problem

import (
	appJudge "github.com/training-judge-center/backend/internal/application/judge"
	appProblem "github.com/training-judge-center/backend/internal/application/problem"
	"github.com/training-judge-center/backend/internal/domain/submission"
)

func toJudgeCandidate(c *appProblem.JudgingCandidate) (*appJudge.JudgingCandidate, error) {
	if c == nil {
		return nil, nil
	}
	out := &appJudge.JudgingCandidate{TestCasesKey: c.TestCasesKey}
	if c.CheckerFileKey != "" {
		lang, err := submission.NewLanguage(c.CheckerLanguage)
		if err != nil {
			return nil, err
		}
		out.Checker = &appJudge.CandidateChecker{FileKey: c.CheckerFileKey, Language: lang, CompiledKey: c.CheckerCompiledKey}
	}
	return out, nil
}
