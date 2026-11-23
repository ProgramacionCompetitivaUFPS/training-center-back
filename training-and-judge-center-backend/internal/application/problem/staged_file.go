package problem

import (
	"context"
	"io"
	"log/slog"

	"github.com/training-judge-center/backend/pkg/apperror"
)

// StagedFile is an upload already received, held outside the process heap.
type StagedFile interface {
	io.ReaderAt
	Size() int64
}

// readStaged loads a small staged file; callers cap limit before the file is this big.
func readStaged(ctx context.Context, f StagedFile, limit int64) ([]byte, error) {
	if f == nil || f.Size() > limit {
		return nil, apperror.NewPayloadTooLarge("File size exceeds allowed limit")
	}
	data, err := io.ReadAll(io.NewSectionReader(f, 0, f.Size()))
	if err != nil {
		slog.ErrorContext(ctx, "staged file: read failed", "error", err)
		return nil, apperror.NewInternal()
	}
	return data, nil
}

func stagedReader(f StagedFile) io.Reader {
	return io.NewSectionReader(f, 0, f.Size())
}
