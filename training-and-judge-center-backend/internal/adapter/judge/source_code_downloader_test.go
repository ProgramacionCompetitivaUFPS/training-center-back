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

func TestSourceCodeDownloader_Download_ReturnsTheStoredBytes(t *testing.T) {
	var requested string
	d := &SourceCodeDownloader{reader: &mockGCSReader{
		readObjectFn: func(_ context.Context, object string) (io.ReadCloser, error) {
			requested = object
			return io.NopCloser(strings.NewReader("int main(){}")), nil
		},
	}}

	code, err := d.Download(context.Background(), "submissions/s1/code.cpp")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(code) != "int main(){}" {
		t.Errorf("code: got %q, want %q", code, "int main(){}")
	}
	if requested != "submissions/s1/code.cpp" {
		t.Errorf("object requested: got %q, want %q", requested, "submissions/s1/code.cpp")
	}
}

func TestSourceCodeDownloader_Download_MapsStorageFailures(t *testing.T) {
	tests := []struct {
		name     string
		reader   *mockGCSReader
		wantKind apperror.Kind
	}{
		{"a missing object is not found", readerReturning(nil, storage.ErrObjectNotExist), apperror.KindNotFound},
		{"a storage error is internal", readerReturning(nil, errors.New("network error")), apperror.KindInternal},
		{"a read that breaks midway is internal", readerReturning(failingReader{}, nil), apperror.KindInternal},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &SourceCodeDownloader{reader: tt.reader}

			code, err := d.Download(context.Background(), "submissions/s1/code.cpp")

			assertAppErrorKind(t, err, tt.wantKind)
			if code != nil {
				t.Errorf("expected no code alongside an error, got %q", code)
			}
		})
	}
}
