package judge

import (
	"context"

	"cloud.google.com/go/storage"
)

type gcsWriter interface {
	writeObject(ctx context.Context, object string, content []byte) error
}

type gcsClientWriter struct {
	client *storage.Client
	bucket string
}

func (w *gcsClientWriter) writeObject(ctx context.Context, object string, content []byte) error {
	wc := w.client.Bucket(w.bucket).Object(object).NewWriter(ctx)
	if _, err := wc.Write(content); err != nil {
		wc.Close()
		return err
	}
	return wc.Close()
}

func newGCSWriter(client *storage.Client, bucket string) gcsWriter {
	return &gcsClientWriter{client: client, bucket: bucket}
}
