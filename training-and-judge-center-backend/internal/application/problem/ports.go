package problem

import (
	"context"

	"github.com/training-judge-center/backend/internal/domain/user"
)

type UserProvider interface {
	ExistsByID(ctx context.Context, userID string) (bool, error)
	GetDisplay(ctx context.Context, userID string) (*user.Display, error)
	GetDisplays(ctx context.Context, userIDs []string) (map[string]*user.Display, error)
}
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
