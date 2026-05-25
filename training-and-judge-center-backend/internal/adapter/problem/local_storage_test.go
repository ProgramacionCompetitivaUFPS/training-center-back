package problem_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/training-judge-center/backend/internal/adapter/problem"
)

func TestLocalFileRepository_Validate_CreatesDir(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "storage")
	repo := problem.NewLocalFileRepository(dir)

	if err := repo.Validate(); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if _, err := os.Stat(dir); os.IsNotExist(err) {
		t.Fatal("expected directory to be created by Validate()")
	}
}

func TestLocalFileRepository_Validate_WritableDir(t *testing.T) {
	dir := t.TempDir()
	repo := problem.NewLocalFileRepository(dir)

	if err := repo.Validate(); err != nil {
		t.Fatalf("expected no error for writable dir, got: %v", err)
	}
}

func TestLocalFileRepository_Validate_UnwritableDir(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("chmod 0000 does not block writes on Windows — ACLs, not POSIX permissions")
	}
	if os.Getuid() == 0 {
		t.Skip("root bypasses permission checks")
	}

	parent := t.TempDir()
	dir := filepath.Join(parent, "storage")
	if err := os.MkdirAll(dir, 0000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(dir, 0755) })

	repo := problem.NewLocalFileRepository(dir)
	if err := repo.Validate(); err == nil {
		t.Fatal("expected error for unwritable dir, got nil")
	}
}
