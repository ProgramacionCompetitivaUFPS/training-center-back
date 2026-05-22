package problem

import "context"

type ParsedFile struct {
	Path    string
	Content []byte
}

type ZipParser interface {
	ParseTestCasesZip(ctx context.Context, zipData []byte) ([]ParsedFile, error)
}
