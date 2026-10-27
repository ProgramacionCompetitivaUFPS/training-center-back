package problem

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/training-judge-center/backend/pkg/apperror"
)

const casesZipKey = "problems/test-problem/testcases/xyz/testcases.zip"

// buildZip writes entries in the given order (a map would shuffle them).
func buildZip(t *testing.T, entries ...[2]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for _, e := range entries {
		w, err := zw.Create(e[0])
		require.NoError(t, err)
		_, err = w.Write([]byte(e[1]))
		require.NoError(t, err)
	}
	require.NoError(t, zw.Close())
	return buf.Bytes()
}

func storageWithCasesZip(data []byte) *mockFileStorage {
	return &mockFileStorage{downloadFileFn: func(_ context.Context, path string) ([]byte, error) {
		if path == casesZipKey {
			return data, nil
		}
		return nil, apperror.NewNotFound(apperror.ErrCodeNotFound, "file not found")
	}}
}

func appErr(t *testing.T, err error) *apperror.AppError {
	t.Helper()
	var ae *apperror.AppError
	require.True(t, errors.As(err, &ae), "expected an AppError, got %v", err)
	return ae
}

func TestListTestCases_OrdersSamplesFirstAndNumbersNumerically(t *testing.T) {
	zipData := buildZip(t,
		[2]string{"data/secret/10.in", "abc"}, [2]string{"data/secret/10.ans", "abcd"},
		[2]string{"data/secret/2.in", "x"}, [2]string{"data/secret/2.ans", "y"},
		[2]string{"data/sample/1.in", "1 2"}, [2]string{"data/sample/1.ans", "3"},
	)
	uc := NewListTestCasesUseCase(repoWith(newPublishedProblemWithTestCases()), storageWithCasesZip(zipData))

	out, err := uc.Execute(context.Background(), ListTestCasesInput{Slug: testSlug, CurrentUser: asAdmin(authorID)})
	require.NoError(t, err)

	assert.True(t, out.HasTestCases)
	assert.Equal(t, 1, out.SampleCount)
	assert.Equal(t, 2, out.SecretCount)
	require.Len(t, out.Cases, 3)
	assert.Equal(t, "sample", out.Cases[0].Kind)
	assert.Equal(t, "2", out.Cases[1].Name, "numeric order: 2 before 10")
	assert.Equal(t, "10", out.Cases[2].Name)
	assert.EqualValues(t, 3, *out.Cases[2].InputBytes)
	assert.EqualValues(t, 4, *out.Cases[2].AnswerBytes)
}

func TestListTestCases_FindsCasesUnderARootFolder(t *testing.T) {
	zipData := buildZip(t,
		[2]string{"my-problem/data/sample/a.in", "1"}, [2]string{"my-problem/data/sample/a.ans", "2"},
	)
	uc := NewListTestCasesUseCase(repoWith(newPublishedProblemWithTestCases()), storageWithCasesZip(zipData))

	out, err := uc.Execute(context.Background(), ListTestCasesInput{Slug: testSlug, CurrentUser: asAdmin(authorID)})
	require.NoError(t, err)
	require.Len(t, out.Cases, 1)
	assert.Equal(t, "a", out.Cases[0].Name)
}

func TestListTestCases_IgnoresNoiseAndReportsAMissingSide(t *testing.T) {
	zipData := buildZip(t,
		[2]string{"data/sample/1.in", "1"},
		[2]string{"data/secret/readme.txt", "not a case"},
		[2]string{"__MACOSX/data/sample/1.in", "junk"},
		[2]string{"data/other/1.in", "wrong dir"},
	)
	uc := NewListTestCasesUseCase(repoWith(newPublishedProblemWithTestCases()), storageWithCasesZip(zipData))

	out, err := uc.Execute(context.Background(), ListTestCasesInput{Slug: testSlug, CurrentUser: asAdmin(authorID)})
	require.NoError(t, err)
	require.Len(t, out.Cases, 1)
	assert.NotNil(t, out.Cases[0].InputBytes)
	assert.Nil(t, out.Cases[0].AnswerBytes, "an .in without its .ans is shown as incomplete, not hidden")
}

func TestListTestCases_NoTestCasesUploadedIsEmptyNotAnError(t *testing.T) {
	uc := NewListTestCasesUseCase(repoWith(newDraftProblem()), &mockFileStorage{})

	out, err := uc.Execute(context.Background(), ListTestCasesInput{Slug: testSlug, CurrentUser: asAdmin(authorID)})
	require.NoError(t, err)
	assert.False(t, out.HasTestCases)
	assert.NotNil(t, out.Cases)
	assert.Empty(t, out.Cases)
}

func TestTestCaseFiles_OnlyAuthorModifiersAndAdminMaySee(t *testing.T) {
	zipData := buildZip(t, [2]string{"data/sample/1.in", "1"}, [2]string{"data/sample/1.ans", "2"})
	storage := storageWithCasesZip(zipData)

	cases := map[string]struct {
		build func() error
	}{
		"list": {func() error {
			_, err := NewListTestCasesUseCase(repoWith(newPublishedProblemWithTestCases()), storage).
				Execute(context.Background(), ListTestCasesInput{Slug: testSlug, CurrentUser: asContestant("stranger")})
			return err
		}},
		"preview": {func() error {
			_, err := NewGetTestCaseUseCase(repoWith(newPublishedProblemWithTestCases()), storage).
				Execute(context.Background(), GetTestCaseInput{Slug: testSlug, Kind: "sample", Name: "1", CurrentUser: asContestant("stranger")})
			return err
		}},
		"download": {func() error {
			_, err := NewDownloadTestCasesUseCase(repoWithTestCases(), storage).
				Execute(context.Background(), DownloadTestCasesInput{Slug: testSlug, CurrentUser: asContestant("stranger")})
			return err
		}},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			ae := appErr(t, tc.build())
			assert.Equal(t, apperror.KindForbidden, ae.Kind)
			assert.Equal(t, ErrCodeInsufficientPermissions, ae.Code)
		})
	}

	t.Run("a modifier can", func(t *testing.T) {
		_, err := NewListTestCasesUseCase(repoWith(newPublishedProblemWithTestCasesAndModifier()), storage).
			Execute(context.Background(), ListTestCasesInput{Slug: testSlug, CurrentUser: asCoach(modifierID)})
		assert.NoError(t, err)
	})
}

func repoWithTestCases() *mockProblemRepository { return repoWith(newPublishedProblemWithTestCases()) }

func TestGetTestCase_ReturnsTheCase(t *testing.T) {
	zipData := buildZip(t, [2]string{"data/secret/7.in", "5 6\n"}, [2]string{"data/secret/7.ans", "11\n"})
	uc := NewGetTestCaseUseCase(repoWith(newPublishedProblemWithTestCases()), storageWithCasesZip(zipData))

	out, err := uc.Execute(context.Background(), GetTestCaseInput{Slug: testSlug, Kind: "secret", Name: "7", CurrentUser: asAdmin(authorID)})
	require.NoError(t, err)
	assert.Equal(t, "5 6\n", out.Input)
	assert.Equal(t, "11\n", out.Answer)
	assert.False(t, out.InputTruncated)
	assert.False(t, out.AnswerTruncated)
}

func TestGetTestCase_CutsLongFilesAndSaysSo(t *testing.T) {
	long := strings.Repeat("a", TestCasePreviewMaxBytes+500)
	zipData := buildZip(t, [2]string{"data/secret/big.in", long}, [2]string{"data/secret/big.ans", "ok"})
	uc := NewGetTestCaseUseCase(repoWith(newPublishedProblemWithTestCases()), storageWithCasesZip(zipData))

	out, err := uc.Execute(context.Background(), GetTestCaseInput{Slug: testSlug, Kind: "secret", Name: "big", CurrentUser: asAdmin(authorID)})
	require.NoError(t, err)
	assert.True(t, out.InputTruncated)
	assert.Len(t, out.Input, TestCasePreviewMaxBytes)
	assert.False(t, out.AnswerTruncated)
}

func TestGetTestCase_DropsABrokenCharacterAtTheCut(t *testing.T) {
	// "é" is two bytes; a cut after the first one must not leave an invalid sequence behind.
	content := strings.Repeat("a", TestCasePreviewMaxBytes-1) + "é"
	zipData := buildZip(t, [2]string{"data/sample/1.in", content}, [2]string{"data/sample/1.ans", "x"})
	uc := NewGetTestCaseUseCase(repoWith(newPublishedProblemWithTestCases()), storageWithCasesZip(zipData))

	out, err := uc.Execute(context.Background(), GetTestCaseInput{Slug: testSlug, Kind: "sample", Name: "1", CurrentUser: asAdmin(authorID)})
	require.NoError(t, err)
	assert.True(t, out.InputTruncated)
	assert.Len(t, out.Input, TestCasePreviewMaxBytes-1)
	assert.NotContains(t, out.Input, "�")
}

func TestGetTestCase_BadKindUnknownCaseAndMissingZip(t *testing.T) {
	zipData := buildZip(t, [2]string{"data/sample/1.in", "1"}, [2]string{"data/sample/1.ans", "2"})
	storage := storageWithCasesZip(zipData)
	uc := NewGetTestCaseUseCase(repoWith(newPublishedProblemWithTestCases()), storage)

	_, err := uc.Execute(context.Background(), GetTestCaseInput{Slug: testSlug, Kind: "hidden", Name: "1", CurrentUser: asAdmin(authorID)})
	assert.Equal(t, apperror.KindValidation, appErr(t, err).Kind)

	_, err = uc.Execute(context.Background(), GetTestCaseInput{Slug: testSlug, Kind: "secret", Name: "1", CurrentUser: asAdmin(authorID)})
	assert.Equal(t, ErrCodeProblemFileNotFound, appErr(t, err).Code, "1 exists as a sample, not as a secret case")

	noZip := NewGetTestCaseUseCase(repoWith(newDraftProblem()), &mockFileStorage{})
	_, err = noZip.Execute(context.Background(), GetTestCaseInput{Slug: testSlug, Kind: "sample", Name: "1", CurrentUser: asAdmin(authorID)})
	assert.Equal(t, ErrCodeProblemFileNotFound, appErr(t, err).Code)
}

func TestDownloadTestCases_ReturnsTheUploadedZipAsIs(t *testing.T) {
	zipData := buildZip(t, [2]string{"data/sample/1.in", "1"}, [2]string{"data/sample/1.ans", "2"})
	uc := NewDownloadTestCasesUseCase(repoWithTestCases(), storageWithCasesZip(zipData))

	out, err := uc.Execute(context.Background(), DownloadTestCasesInput{Slug: testSlug, CurrentUser: asAdmin(authorID)})
	require.NoError(t, err)
	assert.Equal(t, "test-problem-testcases.zip", out.FileName)
	assert.Equal(t, zipData, out.Data)
}

func TestDownloadTestCases_NothingUploaded(t *testing.T) {
	uc := NewDownloadTestCasesUseCase(repoWith(newDraftProblem()), &mockFileStorage{})
	_, err := uc.Execute(context.Background(), DownloadTestCasesInput{Slug: testSlug, CurrentUser: asAdmin(authorID)})
	assert.Equal(t, ErrCodeProblemFileNotFound, appErr(t, err).Code)
}

func TestTestCaseFiles_MissingStorageObjectIsNotReportedAsMissingProblem(t *testing.T) {
	// The problem says it has test cases but the object is gone: that must read as a broken
	// file, not as the 404 a client would take to mean the problem doesn't exist.
	uc := NewListTestCasesUseCase(repoWithTestCases(), &mockFileStorage{downloadFileFn: func(context.Context, string) ([]byte, error) {
		return nil, apperror.NewNotFound(apperror.ErrCodeNotFound, "file not found")
	}})
	_, err := uc.Execute(context.Background(), ListTestCasesInput{Slug: testSlug, CurrentUser: asAdmin(authorID)})
	ae := appErr(t, err)
	assert.Equal(t, ErrCodeProblemFileNotFound, ae.Code)
	assert.Contains(t, ae.Message, "missing from storage")
}

func TestListTestCases_CorruptStoredZipIsAnInternalError(t *testing.T) {
	uc := NewListTestCasesUseCase(repoWithTestCases(), storageWithCasesZip([]byte("not a zip")))
	_, err := uc.Execute(context.Background(), ListTestCasesInput{Slug: testSlug, CurrentUser: asAdmin(authorID)})
	assert.Equal(t, apperror.KindInternal, appErr(t, err).Kind)
}
