package problem

import "context"

type ProblemFileRepository interface {
	UploadFile(ctx context.Context, path string, content []byte) error
	DeleteFile(ctx context.Context, path string) error
	DeleteFilesWithPrefix(ctx context.Context, prefix string) error
}

type ParsedFile struct {
	Path    string
	Content []byte
}

type ZipParser interface {
	ParseTestCasesZip(zipData []byte) ([]ParsedFile, error)
}

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
	ParsePackageZip(zipData []byte) (*ParsedPackage, error)
}
