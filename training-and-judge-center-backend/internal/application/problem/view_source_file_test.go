package problem

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/training-judge-center/backend/pkg/apperror"
)

func readSource(t *testing.T, fileType, fileName string) (*GetSourceFileOutput, error) {
	t.Helper()
	p, storage := completeProblem(t, "X")
	return NewGetSourceFileUseCase(repoWith(p), storage).
		Execute(context.Background(), GetSourceFileInput{Slug: testSlug, FileType: fileType, FileName: fileName, CurrentUser: asAdmin(authorID)})
}

func TestGetSourceFile_ReturnsEachKindOfFile(t *testing.T) {
	cases := []struct {
		name, fileType, fileName         string
		wantFilename, wantLang, wantBody string
	}{
		{"a solution", "solution", "alt.py", "alt.py", "python310", "print(1)"},
		{"another solution", "solution", "sol.cpp", "sol.cpp", "cpp20", "int main(){}"},
		{"the checker", "checker", "", "my_checker.cpp", "cpp20", "// checker"},
		{"the validator", "validator", "", "val.py", "python310", "# validator"},
		{"the type is case-insensitive, like on upload", "Checker", "", "my_checker.cpp", "cpp20", "// checker"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out, err := readSource(t, tc.fileType, tc.fileName)
			require.NoError(t, err)
			assert.Equal(t, tc.wantFilename, out.Filename, "the name the author uploaded, not the storage key")
			assert.Equal(t, tc.wantLang, out.Language)
			assert.Equal(t, tc.wantBody, out.Content)
			assert.Equal(t, len(tc.wantBody), out.SizeBytes)
			assert.False(t, out.Truncated)
		})
	}
}

func TestGetSourceFile_Rejections(t *testing.T) {
	t.Run("a solution needs its file name", func(t *testing.T) {
		_, err := readSource(t, "solution", "")
		assert.Equal(t, ErrCodeProblemMissingFilename, appErr(t, err).Code)
	})
	t.Run("a solution that isn't there", func(t *testing.T) {
		_, err := readSource(t, "solution", "nope.cpp")
		assert.Equal(t, apperror.KindNotFound, appErr(t, err).Kind)
	})
	t.Run("a checker the problem doesn't have", func(t *testing.T) {
		uc := NewGetSourceFileUseCase(repoWith(newPublishedProblem()), &mockFileStorage{})
		_, err := uc.Execute(context.Background(), GetSourceFileInput{Slug: testSlug, FileType: "checker", CurrentUser: asAdmin(authorID)})
		assert.Equal(t, ErrCodeProblemFileNotFound, appErr(t, err).Code)
	})
	t.Run("unknown type", func(t *testing.T) {
		_, err := readSource(t, "exe", "")
		assert.Equal(t, ErrCodeProblemInvalidFileType, appErr(t, err).Code)
	})
	t.Run("test cases are not source files", func(t *testing.T) {
		_, err := readSource(t, "testCases", "")
		ae := appErr(t, err)
		assert.Equal(t, ErrCodeProblemInvalidFileType, ae.Code)
		assert.Contains(t, ae.Message, "ZIP")
	})
}

func TestGetSourceFile_OnlyAuthorModifiersAndAdminMaySee(t *testing.T) {
	p, storage := completeProblem(t, "X")
	uc := NewGetSourceFileUseCase(repoWith(p), storage)

	_, err := uc.Execute(context.Background(), GetSourceFileInput{Slug: testSlug, FileType: "solution", FileName: "sol.cpp", CurrentUser: asContestant("stranger")})
	ae := appErr(t, err)
	assert.Equal(t, apperror.KindForbidden, ae.Kind)
	assert.NotContains(t, ae.Message, "int main", "a forbidden answer must not carry the code")

	_, err = uc.Execute(context.Background(), GetSourceFileInput{Slug: testSlug, FileType: "solution", FileName: "sol.cpp", CurrentUser: asCoach(authorID)})
	assert.NoError(t, err)
}

func TestGetSourceFile_CutsAVeryLongFileAndSaysSo(t *testing.T) {
	p, storage := completeProblem(t, "X")
	long := strings.Repeat("a", SourceFilePreviewMaxBytes+10)
	good := storage.downloadFileFn
	storage.downloadFileFn = func(ctx context.Context, path string) ([]byte, error) {
		if path == "problems/test-problem/solution/sol.cpp" {
			return []byte(long), nil
		}
		return good(ctx, path)
	}

	out, err := NewGetSourceFileUseCase(repoWith(p), storage).
		Execute(context.Background(), GetSourceFileInput{Slug: testSlug, FileType: "solution", FileName: "sol.cpp", CurrentUser: asAdmin(authorID)})
	require.NoError(t, err)
	assert.True(t, out.Truncated)
	assert.Len(t, out.Content, SourceFilePreviewMaxBytes)
	assert.Equal(t, SourceFilePreviewMaxBytes+10, out.SizeBytes, "the real size is still reported")
}

func TestGetSourceFile_AFileStorageLostIsAMissingFileNotAMissingProblem(t *testing.T) {
	p, storage := completeProblem(t, "X")
	storage.downloadFileFn = func(context.Context, string) ([]byte, error) {
		return nil, apperror.NewNotFound(apperror.ErrCodeNotFound, "file not found")
	}

	_, err := NewGetSourceFileUseCase(repoWith(p), storage).
		Execute(context.Background(), GetSourceFileInput{Slug: testSlug, FileType: "checker", CurrentUser: asAdmin(authorID)})
	ae := appErr(t, err)
	assert.Equal(t, ErrCodeProblemFileNotFound, ae.Code)
	assert.Contains(t, ae.Message, "my_checker.cpp")
	assert.Contains(t, ae.Message, "missing from storage")
}
