package judge

import "context"

type TestCase struct {
	Input          []byte
	ExpectedOutput []byte
}

type TestCaseProvider interface {
	GetTestCases(ctx context.Context, problemID string) ([]TestCase, error)
}
