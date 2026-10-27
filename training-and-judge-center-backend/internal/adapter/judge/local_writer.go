package judge

import (
	"context"
	"os"
	"path/filepath"
)

type localWriter struct {
	dir string
}

func (w *localWriter) writeObject(_ context.Context, object string, content []byte) error {
	path := filepath.Join(w.dir, object)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, content, 0o644)
}

func newLocalWriter(dir string) gcsWriter {
	return &localWriter{dir: dir}
}
