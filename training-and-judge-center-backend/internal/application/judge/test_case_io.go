package judge

import (
	"context"
	"io"
)

// The three helpers below open one file of a case, hand it to the sandbox and close
// it right after, so a case is never in memory whole and never open for longer than its use.

func runCase(ctx context.Context, session ExecutionSession, set TestCaseSet, ref TestCaseRef, timeLimitMs int) (RunResult, error) {
	input, err := set.OpenInput(ref.Name)
	if err != nil {
		return RunResult{}, err
	}
	defer input.Close()
	return session.RunTestCase(ctx, RunRequest{Input: input, TimeLimitMs: timeLimitMs})
}

func checkCase(ctx context.Context, checker CheckerSession, set TestCaseSet, ref TestCaseRef) (CheckResult, error) {
	answer, err := set.OpenAnswer(ref.Name)
	if err != nil {
		return CheckResult{}, err
	}
	defer answer.Close()
	return checker.Check(ctx, answer, ref.AnswerSize)
}

func validateCase(ctx context.Context, validator ValidatorSession, set TestCaseSet, ref TestCaseRef) (ValidatorRunResult, error) {
	input, err := set.OpenInput(ref.Name)
	if err != nil {
		return ValidatorRunResult{}, err
	}
	defer input.Close()
	return validator.Validate(ctx, input, ref.InputSize)
}

// answerPreview reads only the head of the jury's answer, enough for a failure report.
func answerPreview(set TestCaseSet, name string) []byte {
	answer, err := set.OpenAnswer(name)
	if err != nil {
		return nil
	}
	defer answer.Close()
	head, _ := io.ReadAll(io.LimitReader(answer, maxFailurePreviewBytes+1))
	return truncatePreview(head)
}
