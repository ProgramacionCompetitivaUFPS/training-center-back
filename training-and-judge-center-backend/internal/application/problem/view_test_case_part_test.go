package problem

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/training-judge-center/backend/pkg/apperror"
)

func readPart(t *testing.T, zipData []byte, kind, name, part string) ([]byte, error) {
	t.Helper()
	uc := NewGetTestCasePartUseCase(repoWithTestCases(), storageWithCasesZip(zipData))
	return uc.Execute(context.Background(), GetTestCasePartInput{Slug: testSlug, Kind: kind, Name: name, Part: part, CurrentUser: asAdmin(authorID)})
}

func TestGetTestCasePart_ReturnsTheWholeFileEvenPastThePreviewCap(t *testing.T) {
	big := strings.Repeat("123456789 ", TestCasePreviewMaxBytes) // ~640 KB, ten times the preview cap
	zipData := buildZip(t, [2]string{"data/secret/big.in", big}, [2]string{"data/secret/big.ans", "42\n"})

	input, err := readPart(t, zipData, "secret", "big", "input")
	require.NoError(t, err)
	assert.Equal(t, big, string(input), "the full-text endpoint must not cut anything")

	answer, err := readPart(t, zipData, "secret", "big", "answer")
	require.NoError(t, err)
	assert.Equal(t, "42\n", string(answer))
}

func TestGetTestCasePart_KeepsTheBytesAsTheyAre(t *testing.T) {
	// Not UTF-8 and with CRLF: the part is served as stored, the client decides what to do with it.
	raw := "caf\xe9\r\n1 2\r\n"
	zipData := buildZip(t, [2]string{"data/sample/1.in", raw}, [2]string{"data/sample/1.ans", "x"})

	got, err := readPart(t, zipData, "sample", "1", "input")
	require.NoError(t, err)
	assert.Equal(t, raw, string(got))
}

func TestGetTestCasePart_Rejections(t *testing.T) {
	zipData := buildZip(t, [2]string{"data/sample/1.in", "1"}) // no .ans on purpose

	t.Run("bad kind", func(t *testing.T) {
		_, err := readPart(t, zipData, "hidden", "1", "input")
		assert.Equal(t, apperror.KindValidation, appErr(t, err).Kind)
	})
	t.Run("bad part", func(t *testing.T) {
		_, err := readPart(t, zipData, "sample", "1", "output")
		assert.Equal(t, apperror.KindValidation, appErr(t, err).Kind)
	})
	t.Run("unknown case", func(t *testing.T) {
		_, err := readPart(t, zipData, "sample", "9", "input")
		assert.Equal(t, ErrCodeProblemFileNotFound, appErr(t, err).Code)
	})
	t.Run("the side the ZIP lacks", func(t *testing.T) {
		_, err := readPart(t, zipData, "sample", "1", "answer")
		ae := appErr(t, err)
		assert.Equal(t, ErrCodeProblemFileNotFound, ae.Code)
		assert.Contains(t, ae.Message, "answer")
	})
	t.Run("nothing uploaded", func(t *testing.T) {
		uc := NewGetTestCasePartUseCase(repoWith(newDraftProblem()), &mockFileStorage{})
		_, err := uc.Execute(context.Background(), GetTestCasePartInput{Slug: testSlug, Kind: "sample", Name: "1", Part: "input", CurrentUser: asAdmin(authorID)})
		assert.Equal(t, ErrCodeProblemFileNotFound, appErr(t, err).Code)
	})
}

func TestGetTestCasePart_OnlyAuthorModifiersAndAdminMaySee(t *testing.T) {
	zipData := buildZip(t, [2]string{"data/secret/1.in", "5 6"}, [2]string{"data/secret/1.ans", "11"})
	uc := NewGetTestCasePartUseCase(repoWithTestCases(), storageWithCasesZip(zipData))

	got, err := uc.Execute(context.Background(), GetTestCasePartInput{Slug: testSlug, Kind: "secret", Name: "1", Part: "input", CurrentUser: asContestant("stranger")})
	assert.Nil(t, got, "a forbidden answer must not carry the data")
	assert.Equal(t, apperror.KindForbidden, appErr(t, err).Kind)

	_, err = uc.Execute(context.Background(), GetTestCasePartInput{Slug: testSlug, Kind: "secret", Name: "1", Part: "input", CurrentUser: asCoach(authorID)})
	assert.NoError(t, err)
}

func TestGetSourceFile_FullReturnsEverythingPastTheCap(t *testing.T) {
	p, storage := completeProblem(t, "X")
	long := strings.Repeat("a", SourceFilePreviewMaxBytes+5000)
	good := storage.downloadFileFn
	storage.downloadFileFn = func(ctx context.Context, path string) ([]byte, error) {
		if path == "problems/test-problem/solution/sol.cpp" {
			return []byte(long), nil
		}
		return good(ctx, path)
	}
	uc := NewGetSourceFileUseCase(repoWith(p), storage)

	preview, err := uc.Execute(context.Background(), GetSourceFileInput{Slug: testSlug, FileType: "solution", FileName: "sol.cpp", CurrentUser: asAdmin(authorID)})
	require.NoError(t, err)
	assert.True(t, preview.Truncated)

	full, err := uc.Execute(context.Background(), GetSourceFileInput{Slug: testSlug, FileType: "solution", FileName: "sol.cpp", Full: true, CurrentUser: asAdmin(authorID)})
	require.NoError(t, err)
	assert.False(t, full.Truncated)
	assert.Equal(t, long, full.Content)
	assert.Equal(t, len(long), full.SizeBytes)
}
