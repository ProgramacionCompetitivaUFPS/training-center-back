package submission

import "context"

type SourceCodeReader interface {
	Read(ctx context.Context, path string) ([]byte, error)
}
