package problem

import "context"

type ProblemFileRepository interface {
	UploadFile(ctx context.Context, path string, content []byte) error
	DeleteFile(ctx context.Context, path string) error
	DeleteFilesWithPrefix(ctx context.Context, prefix string) error
}
