package problem

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	domainProblem "github.com/training-judge-center/backend/internal/domain/problem"
	"github.com/training-judge-center/backend/pkg/apperror"
)

func newDeleteFileFixture(t *testing.T) (*DeleteProblemFileUseCase, *recordingStorage, *int) {
	t.Helper()
	storage := newRecordingStorage()
	saves := 0
	repo := repoWith(newDraftProblemOwningFiles())
	repo.saveFn = func(context.Context, *domainProblem.Problem) error { saves++; return nil }
	return NewDeleteProblemFileUseCase(repo, &mockValidationRepository{}, storage), storage, &saves
}

func deleteFileInput(fileType, fileName string) DeleteProblemFileInput {
	return DeleteProblemFileInput{Slug: testSlug, FileType: fileType, FileName: fileName, CurrentUser: asCoach(authorID)}
}

func TestDeleteProblemFile_Checker_DropsTheSourceAndTheCompiledBinary(t *testing.T) {
	uc, storage, _ := newDeleteFileFixture(t)

	err := uc.Execute(context.Background(), deleteFileInput(FileTypeChecker, ""))

	assert.NoError(t, err)
	assert.ElementsMatch(t, []string{liveCheckerKey, liveCheckerCompKey}, storage.deletedFiles)
	assert.Empty(t, storage.deletedPrefixes)
}

func TestDeleteProblemFile_Validator_DropsTheSourceAndTheCompiledBinary(t *testing.T) {
	uc, storage, _ := newDeleteFileFixture(t)

	err := uc.Execute(context.Background(), deleteFileInput(FileTypeValidator, ""))

	assert.NoError(t, err)
	assert.ElementsMatch(t, []string{ownedValidatorKey, ownedValidatorCompKey}, storage.deletedFiles)
}

func TestDeleteProblemFile_TestCases_DropsThePrefix(t *testing.T) {
	uc, storage, _ := newDeleteFileFixture(t)

	err := uc.Execute(context.Background(), deleteFileInput(FileTypeTestCases, ""))

	assert.NoError(t, err)
	assert.Equal(t, []string{liveTestCasesKey}, storage.deletedPrefixes)
	assert.Empty(t, storage.deletedFiles)
}

func TestDeleteProblemFile_Solution_DropsOnlyThatSolution(t *testing.T) {
	uc, storage, _ := newDeleteFileFixture(t)

	err := uc.Execute(context.Background(), deleteFileInput(FileTypeSolution, "Alt.py"))

	assert.NoError(t, err)
	assert.Equal(t, []string{ownedAltSolutionKey}, storage.deletedFiles)
	assert.Empty(t, storage.deletedPrefixes)
}

func TestDeleteProblemFile_SavesTheProblemBeforeDroppingFiles(t *testing.T) {
	uc, storage, saves := newDeleteFileFixture(t)
	savedWhenDropping := -1
	storage.deleteFileFn = func(context.Context, string) error { savedWhenDropping = *saves; return nil }

	err := uc.Execute(context.Background(), deleteFileInput(FileTypeChecker, ""))

	assert.NoError(t, err)
	assert.Equal(t, 1, savedWhenDropping, "a failed save must never leave a problem pointing at a deleted file")
}

func TestDeleteProblemFile_SaveFails_NothingIsDroppedFromStorage(t *testing.T) {
	storage := newRecordingStorage()
	repo := repoWith(newDraftProblemOwningFiles())
	repo.saveFn = func(context.Context, *domainProblem.Problem) error { return apperror.NewInternal() }
	uc := NewDeleteProblemFileUseCase(repo, &mockValidationRepository{}, storage)

	err := uc.Execute(context.Background(), deleteFileInput(FileTypeChecker, ""))

	assert.Error(t, err)
	assert.Zero(t, storage.attempts)
}

func TestDeleteProblemFile_StorageFails_ReportsTheErrorAfterSaving(t *testing.T) {
	uc, storage, saves := newDeleteFileFixture(t)
	storage.failEveryDeletion()

	err := uc.Execute(context.Background(), deleteFileInput(FileTypeChecker, ""))

	assert.Equal(t, apperror.KindInternal, appErr(t, err).Kind)
	assert.Equal(t, 1, *saves)
	assert.Equal(t, 2*maxRetries, storage.attempts)
}
