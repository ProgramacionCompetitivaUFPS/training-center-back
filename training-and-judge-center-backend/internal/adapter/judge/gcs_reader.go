package judge

import (
	"context"
	"io"

	"cloud.google.com/go/storage"
)

type gcsReader interface {
	readObject(ctx context.Context, object string) (io.ReadCloser, error)
}

type gcsClientReader struct {
	client *storage.Client
	bucket string
}

func (r *gcsClientReader) readObject(ctx context.Context, object string) (io.ReadCloser, error) {
	return r.client.Bucket(r.bucket).Object(object).NewReader(ctx)
}

func newGCSReader(client *storage.Client, bucket string) gcsReader {
	return &gcsClientReader{client: client, bucket: bucket}
}
