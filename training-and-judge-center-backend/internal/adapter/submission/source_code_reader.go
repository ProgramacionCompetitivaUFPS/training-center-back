package submission

import "context"

type SourceCodeReader struct {
	read func(ctx context.Context, path string) ([]byte, error)
}

func NewSourceCodeReader(read func(ctx context.Context, path string) ([]byte, error)) *SourceCodeReader {
	return &SourceCodeReader{read: read}
}

func (r *SourceCodeReader) Read(ctx context.Context, path string) ([]byte, error) {
	return r.read(ctx, path)
}
