package problem

import (
	"context"

	"github.com/training-judge-center/backend/internal/domain/problem"
)

// dropStorageKeys deletes what a problem no longer owns and returns the keys it could not delete.
// Adapters already log the raw error, so the caller only reports the leftovers.
func dropStorageKeys(ctx context.Context, storage StagedFileCleaner, keys problem.StorageKeys) []string {
	var failed []string

	if keys.TestCasesPrefix != "" {
		prefix := keys.TestCasesPrefix
		if err := withStorageRetries(func() error { return storage.DeleteFilesWithPrefix(ctx, prefix) }); err != nil {
			failed = append(failed, prefix)
		}
	}
	for _, key := range keys.Files {
		if key == "" {
			continue
		}
		key := key
		if err := withStorageRetries(func() error { return storage.DeleteFile(ctx, key) }); err != nil {
			failed = append(failed, key)
		}
	}
	return failed
}
