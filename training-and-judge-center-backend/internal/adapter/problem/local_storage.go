package problem

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	appProblem "github.com/training-judge-center/backend/internal/application/problem"
)

type LocalFileRepository struct {
	baseDir string
}

var _ appProblem.ProblemFileRepository = (*LocalFileRepository)(nil)

func NewLocalFileRepository(baseDir string) (*LocalFileRepository, error) {
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create base directory %s: %w", baseDir, err)
	}
	return &LocalFileRepository{
		baseDir: baseDir,
	}, nil
}

func (r *LocalFileRepository) resolvePath(path string) string {
	return filepath.Join(r.baseDir, path)
}

func (r *LocalFileRepository) UploadFile(ctx context.Context, path string, content []byte) error {
	fullPath := r.resolvePath(path)

	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	if err := os.WriteFile(fullPath, content, 0644); err != nil {
		return fmt.Errorf("failed to write file %s: %w", fullPath, err)
	}

	return nil
}

func (r *LocalFileRepository) DeleteFile(ctx context.Context, path string) error {
	fullPath := r.resolvePath(path)

	if err := os.Remove(fullPath); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to delete file %s: %w", fullPath, err)
	}

	return nil
}

func (r *LocalFileRepository) DeleteFilesWithPrefix(ctx context.Context, prefix string) error {
	fullPath := r.resolvePath(prefix)
	if err := os.RemoveAll(fullPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove directory %s: %w", fullPath, err)
	}
	return nil
}
