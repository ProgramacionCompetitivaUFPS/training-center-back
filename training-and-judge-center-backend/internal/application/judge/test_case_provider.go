package judge

import "context"

// TestCaseProvider opens a problem's test cases. The set must be closed: it holds
// what backs the cases (a cached archive) until then.
type TestCaseProvider interface {
	Open(ctx context.Context, problemID string) (TestCaseSet, error)
	OpenAt(ctx context.Context, testCasesKey string) (TestCaseSet, error)
}
