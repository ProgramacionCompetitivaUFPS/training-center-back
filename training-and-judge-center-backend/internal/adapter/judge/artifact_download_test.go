package judge

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"cloud.google.com/go/storage"
	"github.com/training-judge-center/backend/pkg/apperror"
)

const testArtifactKey = "problems/abc/artifact/compiled"

// failingReader stands in for storage that opens fine and then breaks mid-read.
type failingReader struct{}

func (failingReader) Read([]byte) (int, error) { return 0, errors.New("connection reset") }
func (failingReader) Close() error             { return nil }

func readerReturning(rc io.ReadCloser, err error) *mockGCSReader {
	return &mockGCSReader{
		readObjectFn: func(context.Context, string) (io.ReadCloser, error) { return rc, err },
	}
}

func TestDownloadArtifact_ReturnsTheStoredBytes(t *testing.T) {
	reader := readerReturning(io.NopCloser(strings.NewReader("ELF binary")), nil)

	artifact, err := downloadArtifact(context.Background(), reader, testArtifactKey)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(artifact) != "ELF binary" {
		t.Errorf("artifact: got %q, want %q", artifact, "ELF binary")
	}
}

func TestDownloadArtifact_NotFound_ReturnsInternal(t *testing.T) {
	reader := readerReturning(nil, storage.ErrObjectNotExist)

	_, err := downloadArtifact(context.Background(), reader, testArtifactKey)

	assertAppErrorKind(t, err, apperror.KindInternal)
}

func TestDownloadArtifact_StorageError_ReturnsInternal(t *testing.T) {
	reader := readerReturning(nil, errors.New("network error"))

	_, err := downloadArtifact(context.Background(), reader, testArtifactKey)

	assertAppErrorKind(t, err, apperror.KindInternal)
}

func TestDownloadArtifact_ReadFails_ReturnsInternal(t *testing.T) {
	reader := readerReturning(failingReader{}, nil)

	_, err := downloadArtifact(context.Background(), reader, testArtifactKey)

	assertAppErrorKind(t, err, apperror.KindInternal)
}

// An empty artifact means the upload or the stored key is broken. Letting it
// through would fail on every test case instead of once, here.
func TestDownloadArtifact_Empty_ReturnsInternal(t *testing.T) {
	reader := readerReturning(io.NopCloser(strings.NewReader("")), nil)

	_, err := downloadArtifact(context.Background(), reader, testArtifactKey)

	assertAppErrorKind(t, err, apperror.KindInternal)
}
