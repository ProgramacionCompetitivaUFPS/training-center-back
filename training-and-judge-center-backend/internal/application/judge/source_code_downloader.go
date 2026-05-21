package judge

import "context"

type SourceCodeDownloader interface {
	Download(ctx context.Context, path string) ([]byte, error)
}
