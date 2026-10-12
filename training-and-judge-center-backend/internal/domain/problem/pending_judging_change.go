package problem

import "github.com/training-judge-center/backend/pkg/apperror"

// PendingJudgingChange is a replacement of a published problem's test cases and/or checker, not in effect until validated.
type PendingJudgingChange struct {
	testCasesKey *string
	checker      *JudgingFile
}

func NewPendingJudgingChange(testCasesKey *string, checker *JudgingFile) (PendingJudgingChange, error) {
	if testCasesKey == nil && checker == nil {
		return PendingJudgingChange{}, apperror.NewInternal()
	}
	return PendingJudgingChange{testCasesKey: testCasesKey, checker: checker}, nil
}

func RestorePendingJudgingChange(testCasesKey *string, checker *JudgingFile) PendingJudgingChange {
	return PendingJudgingChange{testCasesKey: testCasesKey, checker: checker}
}

func (c PendingJudgingChange) TestCasesKey() *string { return c.testCasesKey }
func (c PendingJudgingChange) Checker() *JudgingFile { return c.checker }
