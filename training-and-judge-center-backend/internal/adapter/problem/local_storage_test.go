package problem

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/training-judge-center/backend/pkg/apperror"
)

func newLocalRepo(t *testing.T) (*LocalFileRepository, string) {
	t.Helper()
	dir := t.TempDir()
	repo, err := NewLocalFileRepository(filepath.Join(dir, "storage"))
	require.NoError(t, err)
	return repo, dir
}

func TestNewLocalFileRepository_CreatesTheBaseDirectory(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nested", "storage")

	_, err := NewLocalFileRepository(dir)

	require.NoError(t, err)
	info, err := os.Stat(dir)
	require.NoError(t, err)
	assert.True(t, info.IsDir())
}

func TestNewLocalFileRepository_FailsWhenTheBaseIsAFile(t *testing.T) {
	file := filepath.Join(t.TempDir(), "occupied")
	require.NoError(t, os.WriteFile(file, []byte("x"), 0o600))

	_, err := NewLocalFileRepository(filepath.Join(file, "storage"))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create base directory")
}

func TestLocalFileRepository_UploadAndDownload_RoundTripsNestedPaths(t *testing.T) {
	repo, _ := newLocalRepo(t)

	require.NoError(t, repo.UploadFile(context.Background(), "problems/two-sum/tests.zip", []byte("zip bytes")))
	got, err := repo.DownloadFile(context.Background(), "problems/two-sum/tests.zip")

	require.NoError(t, err)
	assert.Equal(t, []byte("zip bytes"), got)
}

func TestLocalFileRepository_UploadFileStream_OverwritesExistingContent(t *testing.T) {
	repo, _ := newLocalRepo(t)
	require.NoError(t, repo.UploadFile(context.Background(), "a.txt", []byte("a much longer first version")))

	require.NoError(t, repo.UploadFileStream(context.Background(), "a.txt", strings.NewReader("short")))

	got, err := repo.DownloadFile(context.Background(), "a.txt")
	require.NoError(t, err)
	assert.Equal(t, []byte("short"), got)
}

type failingReader struct{}

func (failingReader) Read([]byte) (int, error) { return 0, errors.New("source broke") }

func TestLocalFileRepository_UploadFileStream_ReportsAnInternalErrorWhenTheSourceFails(t *testing.T) {
	repo, _ := newLocalRepo(t)

	err := repo.UploadFileStream(context.Background(), "a.txt", failingReader{})

	var appErr *apperror.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, apperror.KindInternal, appErr.Kind)
}

func TestLocalFileRepository_Upload_ReportsAnInternalErrorWhenADirectoryBlocksThePath(t *testing.T) {
	repo, _ := newLocalRepo(t)
	require.NoError(t, repo.UploadFile(context.Background(), "taken", []byte("a file")))

	err := repo.UploadFile(context.Background(), "taken/child.txt", []byte("x"))

	var appErr *apperror.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, apperror.KindInternal, appErr.Kind)
}

func TestLocalFileRepository_DownloadFile_MissingIsNotFound(t *testing.T) {
	repo, _ := newLocalRepo(t)

	_, err := repo.DownloadFile(context.Background(), "nope.txt")

	var appErr *apperror.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, apperror.KindNotFound, appErr.Kind)
	assert.Equal(t, apperror.ErrCodeNotFound, appErr.Code)
}

func TestLocalFileRepository_DownloadFile_ADirectoryIsAnInternalError(t *testing.T) {
	repo, _ := newLocalRepo(t)
	require.NoError(t, repo.UploadFile(context.Background(), "dir/file.txt", []byte("x")))

	_, err := repo.DownloadFile(context.Background(), "dir")

	var appErr *apperror.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, apperror.KindInternal, appErr.Kind)
}

func TestLocalFileRepository_DeleteFile_RemovesItAndIgnoresAMissingOne(t *testing.T) {
	repo, _ := newLocalRepo(t)
	require.NoError(t, repo.UploadFile(context.Background(), "a.txt", []byte("x")))

	require.NoError(t, repo.DeleteFile(context.Background(), "a.txt"))
	require.NoError(t, repo.DeleteFile(context.Background(), "a.txt"))

	_, err := repo.DownloadFile(context.Background(), "a.txt")
	require.Error(t, err)
}

func TestLocalFileRepository_DeleteFile_ANonEmptyDirectoryIsAnInternalError(t *testing.T) {
	repo, _ := newLocalRepo(t)
	require.NoError(t, repo.UploadFile(context.Background(), "dir/file.txt", []byte("x")))

	err := repo.DeleteFile(context.Background(), "dir")

	var appErr *apperror.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, apperror.KindInternal, appErr.Kind)
}

func TestLocalFileRepository_ListFiles_ReturnsSlashPathsRelativeToTheBase(t *testing.T) {
	repo, _ := newLocalRepo(t)
	for _, p := range []string{"problems/a/one.txt", "problems/a/deep/two.txt", "problems/b/three.txt", "other/four.txt"} {
		require.NoError(t, repo.UploadFile(context.Background(), p, []byte("x")))
	}

	got, err := repo.ListFiles(context.Background(), "problems/a")

	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"problems/a/one.txt", "problems/a/deep/two.txt"}, got)

	none, err := repo.ListFiles(context.Background(), "missing/prefix")
	require.NoError(t, err)
	assert.Empty(t, none)
}

func TestLocalFileRepository_DeleteFilesWithPrefix_RemovesTheWholeTree(t *testing.T) {
	repo, _ := newLocalRepo(t)
	require.NoError(t, repo.UploadFile(context.Background(), "problems/a/one.txt", []byte("x")))
	require.NoError(t, repo.UploadFile(context.Background(), "problems/a/deep/two.txt", []byte("x")))
	require.NoError(t, repo.UploadFile(context.Background(), "problems/b/three.txt", []byte("x")))

	require.NoError(t, repo.DeleteFilesWithPrefix(context.Background(), "problems/a"))
	require.NoError(t, repo.DeleteFilesWithPrefix(context.Background(), "problems/a"))

	remaining, err := repo.ListFiles(context.Background(), "problems")
	require.NoError(t, err)
	assert.Equal(t, []string{"problems/b/three.txt"}, remaining)
}
