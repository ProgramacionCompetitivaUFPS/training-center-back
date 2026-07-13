package judge

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"

	"cloud.google.com/go/storage"
)

type localReader struct {
	dir string
}

func (r *localReader) readObject(_ context.Context, object string) (io.ReadCloser, error) {
	f, err := os.Open(filepath.Join(r.dir, object))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, storage.ErrObjectNotExist
		}
		return nil, err
	}
	return f, nil
}

func newLocalReader(dir string) gcsReader {
	return &localReader{dir: dir}
}
