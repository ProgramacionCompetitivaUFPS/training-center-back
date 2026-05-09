package problem

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
