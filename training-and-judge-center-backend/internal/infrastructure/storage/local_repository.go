package storage

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	appProblem "github.com/training-judge-center/backend/internal/application/problem"
)

type LocalStorageRepository struct {
	baseDir string
}

var _ appProblem.ProblemFileRepository = (*LocalStorageRepository)(nil)

func NewLocalStorageRepository(baseDir string) (*LocalStorageRepository, error) {
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create base directory %s: %w", baseDir, err)
	}
	return &LocalStorageRepository{
		baseDir: baseDir,
	}, nil
}

func (r *LocalStorageRepository) resolvePath(path string) string {
	return filepath.Join(r.baseDir, path)
}

func (r *LocalStorageRepository) UploadFile(ctx context.Context, path string, content []byte) error {
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

func (r *LocalStorageRepository) DeleteFile(ctx context.Context, path string) error {
	fullPath := r.resolvePath(path)

	if err := os.Remove(fullPath); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to delete file %s: %w", fullPath, err)
	}

	return nil
}

func (r *LocalStorageRepository) DeleteFilesWithPrefix(ctx context.Context, prefix string) error {
	fullPath := r.resolvePath(prefix)
	if err := os.RemoveAll(fullPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove directory %s: %w", fullPath, err)
	}
	return nil
}
