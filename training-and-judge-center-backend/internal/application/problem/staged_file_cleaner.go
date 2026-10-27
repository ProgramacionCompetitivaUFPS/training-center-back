package problem

import "context"

type StagedFileCleaner interface {
	DeleteFilesWithPrefix(ctx context.Context, prefix string) error
}
