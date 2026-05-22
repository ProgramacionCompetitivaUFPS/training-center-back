package problem

import "context"

type ParsedPackage struct {
	Title       string
	TimeLimitMs *int
	MemoryLimit *int
	Statement   *string
	SampleFiles []ParsedFile
	ZipData     []byte
	Solutions   []ParsedFile
	Checker     *ParsedFile
	Validator   *ParsedFile
}

type ICPCPackageParser interface {
	ParsePackageZip(ctx context.Context, zipData []byte) (*ParsedPackage, error)
}
