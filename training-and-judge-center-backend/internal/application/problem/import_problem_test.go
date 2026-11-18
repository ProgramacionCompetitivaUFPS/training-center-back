package problem

import (
	"bytes"
	"context"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"

	domainProblem "github.com/training-judge-center/backend/internal/domain/problem"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type mockPackageParser struct {
	pkg *ParsedPackage
}

func (m *mockPackageParser) ParsePackageZip(context.Context, StagedFile) (*ParsedPackage, error) {
	return m.pkg, nil
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
