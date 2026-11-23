package problem

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	domainProblem "github.com/training-judge-center/backend/internal/domain/problem"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type mockPackageParser struct {
	pkg *ParsedPackage
	err error
}

func (m *mockPackageParser) ParsePackageZip(context.Context, StagedFile) (*ParsedPackage, error) {
	return m.pkg, m.err
}

func importFixture(storage ProblemFileRepository, saveErr error) *ImportProblemUseCase {
	repo := &mockProblemRepository{
		existsBySlugFn: func(context.Context, domainProblem.Slug) (bool, error) { return false, nil },
		saveFn:         func(context.Context, *domainProblem.Problem) error { return saveErr },
	}
	parser := &mockPackageParser{pkg: &ParsedPackage{
		Title:     "Imported",
		Solutions: []ParsedFile{{Path: "submissions/accepted/Solution.cpp", Content: []byte("int main(){}")}},
		Checker:   &ParsedFile{Path: "output_validators/Checker.cpp", Content: []byte("int main(){}")},
	}}
	return NewImportProblemUseCase(repo, storage, parser, newDefaultSettings())
}

func importInput() ImportProblemInput {
	return ImportProblemInput{Slug: testSlug, Zip: bytes.NewReader([]byte("zip")), CurrentUser: asCoach(authorID)}
}

func captureLogs(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, nil)))
	t.Cleanup(func() { slog.SetDefault(previous) })
	return &buf
}

func TestImportProblem_SaveFails_DropsEverythingItUploaded(t *testing.T) {
	storage := newRecordingStorage()
	uc := importFixture(storage, apperror.NewInternal())

	_, err := uc.Execute(context.Background(), importInput())

	assert.Error(t, err)
	assert.ElementsMatch(t, []string{
		"problems/" + testSlug + "/solution/Solution.cpp",
		"problems/" + testSlug + "/checker/Checker.cpp",
	}, storage.deletedFiles)
}

func TestImportProblem_SaveFailsAndStorageCannotDelete_LogsTheFilesLeftBehind(t *testing.T) {
	logs := captureLogs(t)
	storage := newRecordingStorage()
	storage.failEveryDeletion()
	uc := importFixture(storage, apperror.NewInternal())

	_, err := uc.Execute(context.Background(), importInput())

	assert.Error(t, err)
	assert.Contains(t, logs.String(), "left in storage")
	assert.Contains(t, logs.String(), "problems/"+testSlug+"/solution/Solution.cpp")
	assert.Contains(t, logs.String(), "problems/"+testSlug+"/checker/Checker.cpp")
}

func TestImportProblem_SaveSucceeds_KeepsTheUploadedFilesAndLogsNothing(t *testing.T) {
	logs := captureLogs(t)
	storage := newRecordingStorage()
	uc := importFixture(storage, nil)

	_, err := uc.Execute(context.Background(), importInput())

	assert.NoError(t, err)
	assert.Zero(t, storage.attempts)
	assert.NotContains(t, logs.String(), "left in storage")
}

type uploadLog struct {
	mu       sync.Mutex
	paths    []string
	failWith string
}

func recordUploads(storage *recordingStorage, failSuffix string) *uploadLog {
	log := &uploadLog{failWith: failSuffix}
	storage.uploadFileFn = func(_ context.Context, path string, _ []byte) error {
		log.mu.Lock()
		defer log.mu.Unlock()
		if log.failWith != "" && strings.HasSuffix(path, log.failWith) {
			return apperror.NewInternal()
		}
		log.paths = append(log.paths, path)
		return nil
	}
	return log
}

func (l *uploadLog) withPrefix(prefix string) []string {
	l.mu.Lock()
	defer l.mu.Unlock()
	var found []string
	for _, p := range l.paths {
		if strings.HasPrefix(p, prefix) {
			found = append(found, p)
		}
	}
	return found
}

func fullPackage() *ParsedPackage {
	statement := "A statement"
	timeLimit := 2000
	memoryLimit := 256
	return &ParsedPackage{
		Title:       "Imported",
		Statement:   &statement,
		TimeLimitMs: &timeLimit,
		MemoryLimit: &memoryLimit,
		Zip:         bytes.NewReader([]byte("zip-bytes")),
		SampleFiles: []ParsedFile{
			{Path: "1.in", Content: []byte("1")},
			{Path: "1.ans", Content: []byte("2")},
		},
		Solutions: []ParsedFile{
			{Path: "submissions/accepted/Solution.cpp", Content: []byte("int main(){}")},
			{Path: "submissions/accepted/notes.txt", Content: []byte("ignored")},
		},
		Checker:   &ParsedFile{Path: "output_validators/Checker.cpp", Content: []byte("int main(){}")},
		Validator: &ParsedFile{Path: "input_validators/Validator.cpp", Content: []byte("int main(){}")},
	}
}

func importWith(pkg *ParsedPackage, storage ProblemFileRepository) *ImportProblemUseCase {
	repo := &mockProblemRepository{
		existsBySlugFn: func(context.Context, domainProblem.Slug) (bool, error) { return false, nil },
	}
	return NewImportProblemUseCase(repo, storage, &mockPackageParser{pkg: pkg}, newDefaultSettings())
}

func TestImportProblem_AContestantCannotImport(t *testing.T) {
	storage := newRecordingStorage()
	uc := importWith(fullPackage(), storage)
	input := importInput()
	input.CurrentUser = asContestant(authorID)

	_, err := uc.Execute(context.Background(), input)

	var appErr *apperror.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, apperror.KindForbidden, appErr.Kind)
}

func TestImportProblem_InvalidSlug(t *testing.T) {
	uc := importWith(fullPackage(), newRecordingStorage())
	input := importInput()
	input.Slug = "A"

	_, err := uc.Execute(context.Background(), input)

	assert.Equal(t, []string{"slug: Slug must be at least 3 characters long"}, validationDetails(t, err))
}

func TestImportProblem_SlugAlreadyTaken(t *testing.T) {
	storage := newRecordingStorage()
	log := recordUploads(storage, "")
	repo := &mockProblemRepository{
		existsBySlugFn: func(context.Context, domainProblem.Slug) (bool, error) { return true, nil },
	}
	uc := NewImportProblemUseCase(repo, storage, &mockPackageParser{pkg: fullPackage()}, newDefaultSettings())

	_, err := uc.Execute(context.Background(), importInput())

	var appErr *apperror.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, apperror.KindConflict, appErr.Kind)
	assert.Equal(t, domainProblem.ErrCodeSlugAlreadyExists, appErr.Code)
	assert.Contains(t, appErr.Message, testSlug)
	assert.Empty(t, log.withPrefix(""))
}

func TestImportProblem_ParserFailureIsReturnedAsIs(t *testing.T) {
	parseErr := apperror.NewBadRequest("BAD_ZIP", "the zip is broken")
	repo := &mockProblemRepository{
		existsBySlugFn: func(context.Context, domainProblem.Slug) (bool, error) { return false, nil },
	}
	uc := NewImportProblemUseCase(repo, newRecordingStorage(), &mockPackageParser{err: parseErr}, newDefaultSettings())

	_, err := uc.Execute(context.Background(), importInput())

	assert.Equal(t, parseErr, err)
}

func TestImportProblem_ReportsEveryInvalidPackageFieldInOrder(t *testing.T) {
	storage := newRecordingStorage()
	log := recordUploads(storage, "")
	pkg := fullPackage()
	tooHigh := 999999
	pkg.Title = "   "
	pkg.TimeLimitMs = &tooHigh
	pkg.MemoryLimit = &tooHigh
	uc := importWith(pkg, storage)

	_, err := uc.Execute(context.Background(), importInput())

	assert.Equal(t, []string{
		"title: Title is required",
		"timeLimit: Exceeds global maximum time limit",
		"memoryLimit: Exceeds global maximum memory limit",
	}, validationDetails(t, err))
	assert.Empty(t, log.withPrefix(""))
}

func TestImportProblem_ImportsTheFullPackage(t *testing.T) {
	storage := newRecordingStorage()
	log := recordUploads(storage, "")
	uc := importWith(fullPackage(), storage)

	result, err := uc.Execute(context.Background(), importInput())

	require.NoError(t, err)
	got := result.Problem
	assert.Equal(t, "Imported", got.Title)
	assert.Equal(t, "DRAFT", got.Status)
	require.NotNil(t, got.Statement)
	assert.Equal(t, "A statement", *got.Statement)
	require.NotNil(t, got.TimeLimit)
	assert.Equal(t, 2000, *got.TimeLimit)
	require.NotNil(t, got.MemoryLimit)
	assert.Equal(t, 256, *got.MemoryLimit)

	testCases := log.withPrefix("problems/" + testSlug + "/testcases/")
	require.Len(t, testCases, 3)
	assert.True(t, hasSuffix(testCases, "/testcases.zip"))
	assert.True(t, hasSuffix(testCases, "/1.in"))
	assert.True(t, hasSuffix(testCases, "/1.ans"))
	assert.Len(t, log.paths, 6)
	assert.Contains(t, log.paths, "problems/"+testSlug+"/solution/Solution.cpp")
	assert.Contains(t, log.paths, "problems/"+testSlug+"/checker/Checker.cpp")
	assert.Contains(t, log.paths, "problems/"+testSlug+"/validator/Validator.cpp")
	assert.NotContains(t, log.paths, "problems/"+testSlug+"/solution/notes.txt")
	assert.Zero(t, storage.attempts)
}

func hasSuffix(paths []string, suffix string) bool {
	for _, p := range paths {
		if strings.HasSuffix(p, suffix) {
			return true
		}
	}
	return false
}

func TestImportProblem_TestCaseZipUploadFails_HasNothingToClean(t *testing.T) {
	storage := newRecordingStorage()
	log := recordUploads(storage, "/testcases.zip")
	uc := importWith(fullPackage(), storage)

	_, err := uc.Execute(context.Background(), importInput())

	assert.Error(t, err)
	assert.Zero(t, storage.attempts)
	assert.Empty(t, log.withPrefix(""))
}

func TestImportProblem_SampleUploadFails_DropsTheZipAndEverySampleKey(t *testing.T) {
	storage := newRecordingStorage()
	log := recordUploads(storage, "/1.ans")
	uc := importWith(fullPackage(), storage)

	_, err := uc.Execute(context.Background(), importInput())

	assert.Error(t, err)
	require.Len(t, storage.deletedFiles, 3)
	for _, key := range storage.deletedFiles {
		assert.True(t, strings.HasPrefix(key, "problems/"+testSlug+"/testcases/"), key)
	}
	assert.Empty(t, log.withPrefix("problems/"+testSlug+"/solution/"))
}

func TestImportProblem_SolutionUploadFails_DropsTheTestCaseFilesOnly(t *testing.T) {
	storage := newRecordingStorage()
	recordUploads(storage, "/Solution.cpp")
	uc := importWith(fullPackage(), storage)

	_, err := uc.Execute(context.Background(), importInput())

	assert.Error(t, err)
	require.Len(t, storage.deletedFiles, 3)
	for _, key := range storage.deletedFiles {
		assert.True(t, strings.HasPrefix(key, "problems/"+testSlug+"/testcases/"), key)
	}
}

func TestImportProblem_UnsupportedCheckerExtension_RejectsTheImport(t *testing.T) {
	pkg := fullPackage()
	pkg.Checker = &ParsedFile{Path: "output_validators/Checker.rb", Content: []byte("puts 1")}
	uc := importWith(pkg, newRecordingStorage())

	_, err := uc.Execute(context.Background(), importInput())

	var appErr *apperror.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, apperror.KindBadRequest, appErr.Kind)
	assert.Equal(t, ErrCodeProblemUnsupportedFileExt, appErr.Code)
}

func TestImportProblem_UnsupportedCheckerExtension_LeavesTheEarlierUploadsInStorage(t *testing.T) {
	storage := newRecordingStorage()
	log := recordUploads(storage, "")
	pkg := fullPackage()
	pkg.Checker = &ParsedFile{Path: "output_validators/Checker.rb", Content: []byte("puts 1")}
	uc := importWith(pkg, storage)

	_, err := uc.Execute(context.Background(), importInput())

	assert.Error(t, err)
	assert.NotEmpty(t, log.withPrefix("problems/"+testSlug+"/"))
	assert.Zero(t, storage.attempts, "known gap H13: nothing is cleaned when a verifier extension is unsupported")
}
