package judge

import (
	"context"
	"errors"
	"io"
	"log/slog"

	"cloud.google.com/go/storage"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type SourceCodeDownloader struct {
	reader gcsReader
}

func NewSourceCodeDownloader(client *storage.Client, bucket string) *SourceCodeDownloader {
	return &SourceCodeDownloader{reader: newGCSReader(client, bucket)}
}

func (d *SourceCodeDownloader) Download(ctx context.Context, path string) ([]byte, error) {
	rc, err := d.reader.readObject(ctx, path)
	if err != nil {
		if errors.Is(err, storage.ErrObjectNotExist) {
			slog.ErrorContext(ctx, "source_code_downloader: object not found", "path", path)
			return nil, apperror.NewNotFound(apperror.ErrCodeNotFound, "source code not found")
		}
		slog.ErrorContext(ctx, "source_code_downloader: failed to open object", "path", path, "error", err)
		return nil, apperror.NewInternal()
	}
	defer rc.Close()

	data, err := io.ReadAll(rc)
	if err != nil {
		slog.ErrorContext(ctx, "source_code_downloader: failed to read object", "path", path, "error", err)
		return nil, apperror.NewInternal()
	}
	return data, nil
}
