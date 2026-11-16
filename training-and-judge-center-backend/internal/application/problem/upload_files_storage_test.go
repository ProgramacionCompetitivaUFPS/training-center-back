package problem

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	domainProblem "github.com/training-judge-center/backend/internal/domain/problem"
)

func draftUploadFixture(p *domainProblem.Problem) (*UploadProblemFilesUseCase, *recordingStorage) {
	storage := newRecordingStorage()
	parser := &mockZipParser{
		parseTestCasesZipFn: func(context.Context, []byte) ([]ParsedFile, error) {
			return []ParsedFile{{Path: "data/sample/01.in", Content: []byte("1 2")}}, nil
		},
	}
	uc := NewUploadProblemFilesUseCase(repoWith(p), &mockValidationRepository{}, storage, parser, newCppPythonJavaSettings(), &mockActiveContestChecker{}, &mockAffectedContestsProvider{}, &mockValidationQueue{})
	return uc, storage
}

func draftUploadInput(fileType, fileName string) UploadProblemFilesInput {
	return UploadProblemFilesInput{Slug: testSlug, FileType: fileType, FileName: fileName, FileData: []byte("data"), CurrentUser: asCoach(authorID)}
}

func TestUploadProblemFiles_ReplacingTestCases_DropsTheOldOnesAndKeepsTheNew(t *testing.T) {
	uc, storage := draftUploadFixture(newDraftProblemOwningFiles())

	_, err := uc.Execute(context.Background(), draftUploadInput(FileTypeTestCases, "tc.zip"))

	assert.NoError(t, err)
	assert.Equal(t, []string{liveTestCasesKey}, storage.deletedPrefixes)
	assert.Empty(t, storage.deletedFiles)
}

func TestUploadProblemFiles_ReplacingTheChecker_DropsItsOldCompiledBinary(t *testing.T) {
	uc, storage := draftUploadFixture(newDraftProblemOwningFiles())

	_, err := uc.Execute(context.Background(), draftUploadInput(FileTypeChecker, "Checker.cpp"))

	assert.NoError(t, err)
	assert.Equal(t, []string{liveCheckerCompKey}, storage.deletedFiles, "the source was overwritten in place, only the stale binary goes")
	assert.Empty(t, storage.deletedPrefixes)
}

func TestUploadProblemFiles_ReplacingTheCheckerWithAnotherLanguage_DropsTheOldSourceAndBinary(t *testing.T) {
	uc, storage := draftUploadFixture(newDraftProblemOwningFiles())

	_, err := uc.Execute(context.Background(), draftUploadInput(FileTypeChecker, "Checker.py"))

	assert.NoError(t, err)
	assert.ElementsMatch(t, []string{liveCheckerKey, liveCheckerCompKey}, storage.deletedFiles)
}

func TestUploadProblemFiles_ReplacingTheValidator_DropsItsOldCompiledBinary(t *testing.T) {
	uc, storage := draftUploadFixture(newDraftProblemOwningFiles())

	_, err := uc.Execute(context.Background(), draftUploadInput(FileTypeValidator, "Validator.cpp"))

	assert.NoError(t, err)
	assert.Equal(t, []string{ownedValidatorCompKey}, storage.deletedFiles)
}

func TestUploadProblemFiles_SolutionWithTheSameNameReplacesItInPlace_NothingIsDropped(t *testing.T) {
	uc, storage := draftUploadFixture(newDraftProblemOwningFiles())

	_, err := uc.Execute(context.Background(), draftUploadInput(FileTypeSolution, "Solution.cpp"))

	assert.NoError(t, err)
	assert.Zero(t, storage.attempts, "same name, same key: the upload already overwrote it")
}

func TestUploadProblemFiles_SolutionWithANewName_IsAddedAndNothingIsDropped(t *testing.T) {
	p := newDraftProblemOwningFiles()
	uc, storage := draftUploadFixture(p)

	_, err := uc.Execute(context.Background(), draftUploadInput(FileTypeSolution, "Other.cpp"))

	assert.NoError(t, err)
	assert.Len(t, p.Solutions(), 3)
	assert.Zero(t, storage.attempts)
}

func TestUploadProblemFiles_DropFails_TheUploadStillSucceeds(t *testing.T) {
	uc, storage := draftUploadFixture(newDraftProblemOwningFiles())
	storage.failEveryDeletion()

	_, err := uc.Execute(context.Background(), draftUploadInput(FileTypeChecker, "Checker.cpp"))

	assert.NoError(t, err, "leftovers are logged; the file the user uploaded is in place")
	assert.Equal(t, maxRetries, storage.attempts)
}
