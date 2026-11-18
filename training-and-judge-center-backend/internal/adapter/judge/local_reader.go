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

func (r *localReader) objectSize(_ context.Context, object string) (int64, error) {
	info, err := os.Stat(filepath.Join(r.dir, object))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return 0, storage.ErrObjectNotExist
		}
		return 0, err
	}
	return info.Size(), nil
}

func newSizedLocalReader(dir string) sizedObjectReader {
	return &localReader{dir: dir}
}
