package problem

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"cloud.google.com/go/storage"
	appProblem "github.com/training-judge-center/backend/internal/application/problem"
	"github.com/training-judge-center/backend/pkg/apperror"
	"golang.org/x/sync/errgroup"
	"google.golang.org/api/iterator"
)

type GCSFileRepository struct {
	client *storage.Client
	bucket string
}

var _ appProblem.ProblemFileRepository = (*GCSFileRepository)(nil)

func NewGCSFileRepository(client *storage.Client, bucketName string) *GCSFileRepository {
	return &GCSFileRepository{
		client: client,
		bucket: bucketName,
	}
}

func (r *GCSFileRepository) UploadFile(ctx context.Context, path string, content []byte) error {
	obj := r.client.Bucket(r.bucket).Object(path)
	writer := obj.NewWriter(ctx)

	if _, err := writer.Write(content); err != nil {
		if closeErr := writer.Close(); closeErr != nil {
			slog.ErrorContext(ctx, "GCS writer close failed after write error", "close_error", closeErr, "path", path)
		}
		slog.ErrorContext(ctx, "gcs: failed to write object", "path", path, "error", err)
		return apperror.NewInternal()
	}

	if err := writer.Close(); err != nil {
		slog.ErrorContext(ctx, "gcs: failed to close writer", "path", path, "error", err)
		return apperror.NewInternal()
	}

	return nil
}

func (r *GCSFileRepository) DeleteFile(ctx context.Context, path string) error {
	obj := r.client.Bucket(r.bucket).Object(path)
	if err := obj.Delete(ctx); err != nil {
		if errors.Is(err, storage.ErrObjectNotExist) {
			return nil
		}
		slog.ErrorContext(ctx, "gcs: failed to delete object", "path", path, "error", err)
		return apperror.NewInternal()
	}
	return nil
}

func (r *GCSFileRepository) DeleteFilesWithPrefix(ctx context.Context, prefix string) error {
	it := r.client.Bucket(r.bucket).Objects(ctx, &storage.Query{Prefix: prefix})

	var names []string
	for {
		attrs, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			slog.ErrorContext(ctx, "gcs: failed to list objects", "prefix", prefix, "error", err)
			return apperror.NewInternal()
		}
		names = append(names, attrs.Name)
	}

	if len(names) == 0 {
		return nil
	}

	g, gCtx := errgroup.WithContext(ctx)
	g.SetLimit(10)

	for _, name := range names {
		name := name
		g.Go(func() error {
			if err := r.client.Bucket(r.bucket).Object(name).Delete(gCtx); err != nil {
				if !errors.Is(err, storage.ErrObjectNotExist) {
					slog.ErrorContext(gCtx, "gcs: failed to delete object", "name", name, "error", err)
					return apperror.NewInternal()
				}
			}
			return nil
		})
	}

	return g.Wait()
}

func (r *GCSFileRepository) Validate(ctx context.Context) error {
	_, err := r.client.Bucket(r.bucket).Attrs(ctx)
	if err != nil {
		return fmt.Errorf("GCS storage: bucket %q is not accessible: %w", r.bucket, err)
	}
	return nil
}
