package judge

import "context"

type TestCase struct {
	// Name is e.g. "sample/001" or "secret/002" — JudgeSubmissionUseCase
	// never reads it; ValidateSolutionsUseCase does.
	Name           string
	Input          []byte
	ExpectedOutput []byte
}

type TestCaseProvider interface {
	GetTestCases(ctx context.Context, problemID string) ([]TestCase, error)
}
