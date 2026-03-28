package storage

import (
	"context"
	"fmt"

	"cloud.google.com/go/storage"
	"google.golang.org/api/iterator"
	appProblem "github.com/training-judge-center/backend/internal/application/problem"
)

type GCSProblemFileRepository struct {
	client *storage.Client
	bucket string
}

var _ appProblem.ProblemFileRepository = (*GCSProblemFileRepository)(nil)

func NewGCSProblemFileRepository(client *storage.Client, bucketName string) *GCSProblemFileRepository {
	return &GCSProblemFileRepository{
		client: client,
		bucket: bucketName,
	}
}

func (r *GCSProblemFileRepository) UploadFile(ctx context.Context, path string, content []byte) error {
	obj := r.client.Bucket(r.bucket).Object(path)
	writer := obj.NewWriter(ctx)

	if _, err := writer.Write(content); err != nil {
		_ = writer.Close()
		return fmt.Errorf("failed to write data to GCS object %s: %w", path, err)
	}

	if err := writer.Close(); err != nil {
		return fmt.Errorf("failed to close GCS writer for object %s: %w", path, err)
	}

	return nil
}

func (r *GCSProblemFileRepository) DeleteFile(ctx context.Context, path string) error {
	obj := r.client.Bucket(r.bucket).Object(path)
	if err := obj.Delete(ctx); err != nil {
		if err == storage.ErrObjectNotExist {
			return nil
		}
		return fmt.Errorf("failed to delete GCS object %s: %w", path, err)
	}
	return nil
}

func (r *GCSProblemFileRepository) DeleteFilesWithPrefix(ctx context.Context, prefix string) error {
	it := r.client.Bucket(r.bucket).Objects(ctx, &storage.Query{Prefix: prefix})
	for {
		attrs, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to list GCS objects with prefix %s: %w", prefix, err)
		}
		if err := r.client.Bucket(r.bucket).Object(attrs.Name).Delete(ctx); err != nil {
			if err != storage.ErrObjectNotExist {
				return fmt.Errorf("failed to delete GCS object %s: %w", attrs.Name, err)
			}
		}
	}
	return nil
}
