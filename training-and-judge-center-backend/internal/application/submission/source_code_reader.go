package submission

import "context"

// SourceCodeReader reads a submission's stored source. It returns a NotFound
// apperror with ErrCodeSourceCodeNotFound when the file no longer exists.
type SourceCodeReader interface {
	Read(ctx context.Context, path string) ([]byte, error)
}
