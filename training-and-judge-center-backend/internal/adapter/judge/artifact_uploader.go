package judge

import (
	"context"
	"log/slog"

	"cloud.google.com/go/storage"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type ArtifactUploader struct {
	writer gcsWriter
}

func NewArtifactUploader(client *storage.Client, bucket string) *ArtifactUploader {
	return &ArtifactUploader{writer: newGCSWriter(client, bucket)}
}

func (u *ArtifactUploader) Upload(ctx context.Context, path string, content []byte) error {
	if err := u.writer.writeObject(ctx, path, content); err != nil {
		slog.ErrorContext(ctx, "artifact_uploader: failed to write object", "path", path, "error", err)
		return apperror.NewInternal()
	}
	return nil
}
