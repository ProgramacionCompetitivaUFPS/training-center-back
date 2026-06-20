package submission

import "context"

// SourceStorage adapts any file-upload backend to the application's SourceStorage port.
// The upload and delete funcs are injected at composition time (either local FS or GCS).
type SourceStorage struct {
	upload func(ctx context.Context, path string, data []byte) error
	delete func(ctx context.Context, path string) error
}

func NewSourceStorage(
	upload func(ctx context.Context, path string, data []byte) error,
	delete func(ctx context.Context, path string) error,
) *SourceStorage {
	return &SourceStorage{upload: upload, delete: delete}
}

func (s *SourceStorage) Upload(ctx context.Context, path string, data []byte) error {
	return s.upload(ctx, path, data)
}

func (s *SourceStorage) Delete(ctx context.Context, path string) error {
	return s.delete(ctx, path)
}
