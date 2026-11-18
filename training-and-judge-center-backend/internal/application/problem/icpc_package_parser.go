package problem

import "context"

type ParsedPackage struct {
	Title       string
	TimeLimitMs *int
	MemoryLimit *int
	Statement   *string
	SampleFiles []ParsedFile
	// Zip is the uploaded package itself, kept as the test case archive; nil when it holds no test cases.
	Zip       StagedFile
	Solutions []ParsedFile
	Checker   *ParsedFile
	Validator *ParsedFile
}

type ICPCPackageParser interface {
	ParsePackageZip(ctx context.Context, zip StagedFile) (*ParsedPackage, error)
}
