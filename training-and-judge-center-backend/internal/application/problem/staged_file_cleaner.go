package problem

import "context"

type StagedFileCleaner interface {
	DeleteFile(ctx context.Context, path string) error
	DeleteFilesWithPrefix(ctx context.Context, prefix string) error
}
