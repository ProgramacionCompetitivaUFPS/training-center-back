package judge

import (
	"context"
	"errors"
	"io"
	"log/slog"

	"cloud.google.com/go/storage"

	"github.com/training-judge-center/backend/pkg/apperror"
)

// downloadArtifact reads a compiled checker or validator from storage. The path
// says which one, so callers need no log prefix of their own.
func downloadArtifact(ctx context.Context, reader gcsReader, objectPath string) ([]byte, error) {
	rc, err := reader.readObject(ctx, objectPath)
	if err != nil {
		if errors.Is(err, storage.ErrObjectNotExist) {
			slog.ErrorContext(ctx, "artifact_download: compiled artifact not found", "path", objectPath)
		} else {
			slog.ErrorContext(ctx, "artifact_download: failed to open the compiled artifact", "path", objectPath, "error", err)
		}
		return nil, apperror.NewInternal()
	}
	defer rc.Close()

	artifact, err := io.ReadAll(&io.LimitedReader{R: rc, N: maxArtifactBytes})
	if err != nil {
		slog.ErrorContext(ctx, "artifact_download: failed to read the compiled artifact", "path", objectPath, "error", err)
		return nil, apperror.NewInternal()
	}
	// An empty artifact would fail on every test case instead of here.
	if len(artifact) == 0 {
		slog.ErrorContext(ctx, "artifact_download: the compiled artifact is empty", "path", objectPath)
		return nil, apperror.NewInternal()
	}
	return artifact, nil
}
