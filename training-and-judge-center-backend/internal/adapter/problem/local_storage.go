package problem

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	appProblem "github.com/training-judge-center/backend/internal/application/problem"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type LocalFileRepository struct {
	baseDir string
}

var _ appProblem.ProblemFileRepository = (*LocalFileRepository)(nil)

func NewLocalFileRepository(baseDir string) *LocalFileRepository {
	return &LocalFileRepository{baseDir: baseDir}
}

func (r *LocalFileRepository) Validate() error {
	if err := os.MkdirAll(r.baseDir, 0755); err != nil {
		return fmt.Errorf("local storage: cannot create base directory %s: %w", r.baseDir, err)
	}
	testFile := filepath.Join(r.baseDir, ".write-test")
	if err := os.WriteFile(testFile, []byte("ok"), 0644); err != nil {
		return fmt.Errorf("local storage: base directory is not writable: %w", err)
	}
	os.Remove(testFile)
	return nil
}

func (r *LocalFileRepository) resolvePath(path string) string {
	return filepath.Join(r.baseDir, path)
}

func (r *LocalFileRepository) UploadFile(ctx context.Context, path string, content []byte) error {
	fullPath := r.resolvePath(path)

	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		slog.ErrorContext(ctx, "local: failed to create directory", "dir", dir, "error", err)
		return apperror.NewInternal()
	}

	if err := os.WriteFile(fullPath, content, 0644); err != nil {
		slog.ErrorContext(ctx, "local: failed to write file", "path", fullPath, "error", err)
		return apperror.NewInternal()
	}

	return nil
}

func (r *LocalFileRepository) DeleteFile(ctx context.Context, path string) error {
	fullPath := r.resolvePath(path)

	if err := os.Remove(fullPath); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		slog.ErrorContext(ctx, "local: failed to delete file", "path", fullPath, "error", err)
		return apperror.NewInternal()
	}

	return nil
}

func (r *LocalFileRepository) DeleteFilesWithPrefix(ctx context.Context, prefix string) error {
	fullPath := r.resolvePath(prefix)
	if err := os.RemoveAll(fullPath); err != nil && !os.IsNotExist(err) {
		slog.ErrorContext(ctx, "local: failed to remove directory", "path", fullPath, "error", err)
		return apperror.NewInternal()
	}
	return nil
}
