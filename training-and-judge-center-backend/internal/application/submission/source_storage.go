package submission

import "context"

type SourceStorage interface {
	Upload(ctx context.Context, path string, data []byte) error
	Delete(ctx context.Context, path string) error
}
