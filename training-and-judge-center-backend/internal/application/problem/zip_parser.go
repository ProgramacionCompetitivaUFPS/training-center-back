package problem

type ParsedFile struct {
	Path    string
	Content []byte
}

type ZipParser interface {
	ParseTestCasesZip(zipData []byte) ([]ParsedFile, error)
}
