//go:build integration

package judge

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	"cloud.google.com/go/storage"
)

func TestLocalReader_ReadObject_ReturnsTheFileContent(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "problems/p1/checker.cpp"), "int main(){}")
	r := &localReader{dir: dir}

	rc, err := r.readObject(context.Background(), "problems/p1/checker.cpp")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer rc.Close()
	got, _ := io.ReadAll(rc)

	if string(got) != "int main(){}" {
		t.Errorf("content: got %q, want %q", got, "int main(){}")
	}
}

func TestLocalReader_MissingObject_IsObjectNotExist(t *testing.T) {
	r := &localReader{dir: t.TempDir()}

	_, readErr := r.readObject(context.Background(), "absent.cpp")
	_, sizeErr := r.objectSize(context.Background(), "absent.cpp")

	if !errors.Is(readErr, storage.ErrObjectNotExist) {
		t.Errorf("readObject: got %v, want storage.ErrObjectNotExist", readErr)
	}
	if !errors.Is(sizeErr, storage.ErrObjectNotExist) {
		t.Errorf("objectSize: got %v, want storage.ErrObjectNotExist", sizeErr)
	}
}

func TestLocalReader_ObjectSize_IsTheFileLength(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "tests.zip"), "12345678")
	r := &localReader{dir: dir}

	size, err := r.objectSize(context.Background(), "tests.zip")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if size != 8 {
		t.Errorf("size: got %d, want 8", size)
	}
}

func TestLocalReader_PathThroughAFile_IsNotReportedAsMissing(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "plain.txt"), "x")
	r := &localReader{dir: dir}

	_, readErr := r.readObject(context.Background(), "plain.txt/inner")
	_, sizeErr := r.objectSize(context.Background(), "plain.txt/inner")

	if readErr == nil || errors.Is(readErr, storage.ErrObjectNotExist) {
		t.Errorf("readObject: got %v, want a raw error that is not ErrObjectNotExist", readErr)
	}
	if sizeErr == nil || errors.Is(sizeErr, storage.ErrObjectNotExist) {
		t.Errorf("objectSize: got %v, want a raw error that is not ErrObjectNotExist", sizeErr)
	}
}

func TestLocalWriter_WriteObject_CreatesParentDirectoriesAndTheFile(t *testing.T) {
	dir := t.TempDir()
	w := &localWriter{dir: dir}

	err := w.writeObject(context.Background(), "problems/p1/checker/compiled", []byte("ELF"))

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	path := filepath.Join(dir, "problems/p1/checker/compiled")
	got, readErr := os.ReadFile(path)
	if readErr != nil || string(got) != "ELF" {
		t.Fatalf("file content: got %q, err %v, want %q", got, readErr, "ELF")
	}
	info, _ := os.Stat(path)
	if info.Mode().Perm() != 0o644 {
		t.Errorf("file mode: got %o, want 644", info.Mode().Perm())
	}
	dirInfo, _ := os.Stat(filepath.Dir(path))
	if dirInfo.Mode().Perm() != 0o755 {
		t.Errorf("directory mode: got %o, want 755", dirInfo.Mode().Perm())
	}
}

func TestLocalWriter_WriteObject_ThenReadObject_RoundTrips(t *testing.T) {
	dir := t.TempDir()
	w := &localWriter{dir: dir}
	r := &localReader{dir: dir}

	if err := w.writeObject(context.Background(), "a/b.bin", []byte("payload")); err != nil {
		t.Fatalf("write: %v", err)
	}
	rc, err := r.readObject(context.Background(), "a/b.bin")
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	defer rc.Close()
	got, _ := io.ReadAll(rc)

	if string(got) != "payload" {
		t.Errorf("round trip: got %q, want %q", got, "payload")
	}
}

func TestLocalWriter_WriteObject_UnderAFile_ReturnsAnError(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "plain.txt"), "x")
	w := &localWriter{dir: dir}

	err := w.writeObject(context.Background(), "plain.txt/inner/file", []byte("data"))

	if err == nil {
		t.Error("expected an error when a parent path component is a regular file")
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
